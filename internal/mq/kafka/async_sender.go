package kafka

import (
	"context"
	"sync"
	"time"

	"go-apiadmin/internal/logging"
	"go-apiadmin/internal/metrics"

	kafkaGo "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// AccessAsyncSender 有界异步发送 + 批量聚合：多 worker，从 channel 取消息聚合后写 Kafka。
// 批量触发条件：达到 maxBatch 或等待超过 maxWait。
// 丢弃策略：队列满直接丢 (metrics.HTTPAccessKafkaEnqueue result="dropped").
// 已实现：批量 WriteMessages + 失败降级逐条重试。
// 新增：sync.Pool 复用 messages / spans slice，降低分配与 GC。
// TODO(next): 按字节上限 flush、滞留时间观测、降级落盘。

type AccessAsyncSender struct {
	producer *Producer
	logger   *logging.Logger
	queue    chan AsyncMessage
	workers  int
	wg       sync.WaitGroup
	stopCh   chan struct{}

	maxBatch int
	maxWait  time.Duration

	msgPool  sync.Pool // *[]kafkaGo.Message
	spanPool sync.Pool // *[]trace.Span
}

func NewAccessAsyncSender(p *Producer, l *logging.Logger, queueSize, workers, maxBatch int, maxWait time.Duration) *AccessAsyncSender {
	if queueSize <= 0 {
		queueSize = 10000
	}
	if workers <= 0 {
		workers = 1
	}
	if maxBatch <= 0 {
		maxBatch = 50
	}
	if maxWait <= 0 {
		maxWait = 20 * time.Millisecond
	}
	s := &AccessAsyncSender{
		producer: p,
		logger:   l,
		queue:    make(chan AsyncMessage, queueSize),
		workers:  workers,
		stopCh:   make(chan struct{}),
		maxBatch: maxBatch,
		maxWait:  maxWait,
	}
	// 初始化对象池（延迟引用 s.maxBatch）
	s.msgPool.New = func() any { b := make([]kafkaGo.Message, 0, s.maxBatch); return &b }
	s.spanPool.New = func() any { b := make([]trace.Span, 0, s.maxBatch); return &b }
	return s
}

func (s *AccessAsyncSender) Start() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			batch := make([]AsyncMessage, 0, s.maxBatch) // 长期复用当前 worker 的 batch 容器
			var timer *time.Timer
			var timerCh <-chan time.Time
			flush := func(reason string) {
				if len(batch) == 0 {
					return
				}
				start := time.Now()
				// 滞留时间统计: 计算当前 batch 中 (flush 时刻 - EnqueueAt) 的平均与最大
				var totalDelay time.Duration
				var maxDelay time.Duration
				flushNow := start
				for _, bm := range batch {
					if !bm.EnqueueAt.IsZero() {
						d := flushNow.Sub(bm.EnqueueAt)
						totalDelay += d
						if d > maxDelay {
							maxDelay = d
						}
					}
				}
				if len(batch) > 0 {
					avgDelay := float64(totalDelay.Microseconds()) / 1e6 / float64(len(batch))
					metrics.HTTPAccessKafkaQueueDelayAvg.Observe(avgDelay)
					metrics.HTTPAccessKafkaQueueDelayMax.Observe(float64(maxDelay.Microseconds()) / 1e6)
				}
				// ===== 取出复用切片 =====
				msgsPtr := s.msgPool.Get().(*[]kafkaGo.Message)
				spansPtr := s.spanPool.Get().(*[]trace.Span)
				msgs := (*msgsPtr)[:0]
				spans := (*spansPtr)[:0]
				// 构造 batch 消息 + span
				for _, m := range batch {
					ctxSpan, span := s.producer.startSpan(m.Ctx)
					var hs []kafkaGo.Header
					if len(m.Headers) > 0 {
						hs = make([]kafkaGo.Header, 0, len(m.Headers))
						for k, v := range m.Headers {
							hs = append(hs, kafkaGo.Header{Key: k, Value: []byte(v)})
						}
					}
					hs = s.producer.injectHeaders(ctxSpan, hs)
					msgs = append(msgs, kafkaGo.Message{Key: m.Key, Value: m.Value, Time: time.Now(), Headers: hs})
					spans = append(spans, span)
				}
				writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				err := s.producer.Writer.WriteMessages(writeCtx, msgs...)
				cancel()
				if err != nil {
					for _, sp := range spans {
						sp.SetStatus(codes.Error, err.Error())
						sp.RecordError(err)
						sp.End()
					}
					metrics.HTTPAccessKafkaErrors.Add(float64(len(batch)))
					// 逐条回退（不再创建新的 span，避免重复）
					for _, msg := range batch {
						if len(msg.Headers) > 0 {
							_ = s.producer.SendWithHeaders(msg.Ctx, msg.Key, msg.Value, msg.Headers)
						} else {
							_ = s.producer.Send(msg.Ctx, msg.Key, msg.Value)
						}
					}
				} else {
					for _, sp := range spans {
						sp.End()
					}
				}
				elapsed := time.Since(start)
				metrics.HTTPAccessKafkaBatchFlushTotal.WithLabelValues(reason).Inc()
				metrics.HTTPAccessKafkaBatchSize.Observe(float64(len(batch)))
				metrics.HTTPAccessKafkaSendDuration.Observe(elapsed.Seconds())
				metrics.HTTPAccessKafkaFlushDuration.WithLabelValues(reason).Observe(elapsed.Seconds())
				// 复位 worker batch
				batch = batch[:0]
				// 放回池（限制容量，过大则丢弃）
				if cap(msgs) <= s.maxBatch*2 {
					*msgsPtr = msgs[:0]
					s.msgPool.Put(msgsPtr)
				}
				if cap(spans) <= s.maxBatch*2 {
					*spansPtr = spans[:0]
					s.spanPool.Put(spansPtr)
				}
				if timer != nil {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timerCh = nil
				}
			}
			for {
				select {
				case <-s.stopCh:
					flush("shutdown")
					return
				case msg := <-s.queue:
					metrics.HTTPAccessKafkaQueueDepth.Dec()
					batch = append(batch, msg)
					if len(batch) == 1 { // 启动计时器
						if timer == nil {
							timer = time.NewTimer(s.maxWait)
						} else {
							if !timer.Stop() {
								select {
								case <-timer.C:
								default:
								}
							}
							timer.Reset(s.maxWait)
						}
						timerCh = timer.C
					}
					if len(batch) >= s.maxBatch {
						flush("size")
					}
				case <-timerCh:
					flush("timeout")
				}
			}
		}()
	}
}

// SpanWrap 简单封装便于后续扩展
type SpanWrap struct {
	span interface {
		End(...trace.SpanOption)
		SetStatus(codes.Code, string)
		RecordError(error)
	}
}

// Enqueue 非阻塞放入，满则丢弃
func (s *AccessAsyncSender) Enqueue(m AsyncMessage) {
	select {
	case s.queue <- m:
		metrics.HTTPAccessKafkaEnqueue.WithLabelValues("ok").Inc()
		metrics.HTTPAccessKafkaQueueDepth.Inc()
	default:
		metrics.HTTPAccessKafkaEnqueue.WithLabelValues("dropped").Inc()
	}
}

// Close 停止并尽量消费完队列（graceful）
func (s *AccessAsyncSender) Close(ctx context.Context) error {
	close(s.stopCh)
	close(s.queue)
	s.wg.Wait()
	return nil
}
