package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"

	"github.com/gin-gonic/gin"
)

type FieldsHandler struct{ d Dependencies }

func NewFieldsHandler(d Dependencies) *FieldsHandler { return &FieldsHandler{d: d} }

func (h *FieldsHandler) Index(c *gin.Context) {
	page, limit := pageLimit(c)
	res, err := h.d.Fields.List(c.Request.Context(), service.ListFieldsParams{Hash: c.Query("hash"), Type: int8(qInt(c, "type", 0)), Page: page, Limit: limit})
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, res)
}
func (h *FieldsHandler) Request(c *gin.Context)  { c.Set("_fields_type", int8(0)); h.Index(c) }
func (h *FieldsHandler) Response(c *gin.Context) { c.Set("_fields_type", int8(1)); h.Index(c) }
func (h *FieldsHandler) Add(c *gin.Context) {
	var req struct {
		FieldName, Hash, Default, Range, Info, ShowName string
		DataType, IsMust, Type                          int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	id, err := h.d.Fields.Add(c.Request.Context(), service.AddFieldParams{FieldName: req.FieldName, Hash: req.Hash, Default: req.Default, Range: req.Range, Info: req.Info, Type: req.Type, ShowName: req.ShowName, DataType: req.DataType, IsMust: req.IsMust})
	if err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}
func (h *FieldsHandler) Edit(c *gin.Context) {
	var req struct {
		ID                                              int64
		FieldName, Hash, Default, Range, Info, ShowName *string
		DataType, IsMust, Type                          *int8
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.Fields.Edit(c.Request.Context(), service.EditFieldParams{ID: req.ID, FieldName: req.FieldName, Hash: req.Hash, Default: req.Default, Range: req.Range, Info: req.Info, ShowName: req.ShowName, DataType: req.DataType, IsMust: req.IsMust, Type: req.Type}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *FieldsHandler) Delete(c *gin.Context) {
	id := qInt64(c, "id")
	if err := h.d.Fields.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *FieldsHandler) Upload(c *gin.Context) {
	var req struct {
		Hash string `form:"hash" json:"hash"`
		Type int8   `form:"type" json:"type"`
		JSON string `form:"json" json:"json"`
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if err := h.d.Fields.BatchUpload(c.Request.Context(), service.BatchUploadParams{Hash: req.Hash, Type: req.Type, JSON: req.JSON}); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}
