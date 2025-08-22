/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19-11.7.2-MariaDB, for Win64 (AMD64)
--
-- Host: 127.0.0.1    Database: apiadmin1
-- ------------------------------------------------------
-- Server version	5.7.26

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*M!100616 SET @OLD_NOTE_VERBOSITY=@@NOTE_VERBOSITY, NOTE_VERBOSITY=0 */;

--
-- Table structure for table `admin_auth_group`
--

DROP TABLE IF EXISTS `admin_auth_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_auth_group` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) DEFAULT '' COMMENT '组名称',
  `description` text COMMENT '组描述',
  `status` tinyint(4) DEFAULT '1' COMMENT '组状态：为1正常，为0禁用',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='权限组';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_auth_group`
--

LOCK TABLES `admin_auth_group` WRITE;
/*!40000 ALTER TABLE `admin_auth_group` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_auth_group` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_app_group`
--

DROP TABLE IF EXISTS `admin_app_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_app_group` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(128) DEFAULT '' COMMENT '组名称',
  `description` text COMMENT '组说明',
  `status` tinyint(4) DEFAULT '1' COMMENT '组状态：0表示禁用，1表示启用',
  `hash` varchar(128) DEFAULT '' COMMENT '组标识',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用组，目前只做管理使用，没有实际权限控制';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_app_group`
--

LOCK TABLES `admin_app_group` WRITE;
/*!40000 ALTER TABLE `admin_app_group` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_app_group` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_auth_group_access`
--

DROP TABLE IF EXISTS `admin_auth_group_access`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_auth_group_access` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `uid` int(11) unsigned DEFAULT '0',
  `group_id` varchar(255) DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `uid` (`uid`),
  KEY `group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户和组的对应关系';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_auth_group_access`
--

LOCK TABLES `admin_auth_group_access` WRITE;
/*!40000 ALTER TABLE `admin_auth_group_access` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_auth_group_access` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_user_action`
--

DROP TABLE IF EXISTS `admin_user_action`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_user_action` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `action_name` varchar(50) DEFAULT '' COMMENT '行为名称',
  `uid` int(11) DEFAULT '0' COMMENT '操作用户ID',
  `nickname` varchar(50) DEFAULT '' COMMENT '用户昵称',
  `add_time` int(11) DEFAULT '0' COMMENT '操作时间',
  `data` text COMMENT '用户提交的数据',
  `url` varchar(200) DEFAULT '0' COMMENT '操作URL',
  PRIMARY KEY (`id`),
  KEY `uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户操作日志';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_user_action`
--

LOCK TABLES `admin_user_action` WRITE;
/*!40000 ALTER TABLE `admin_user_action` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_user_action` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_list`
--

DROP TABLE IF EXISTS `admin_list`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_list` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `api_class` varchar(50) DEFAULT '' COMMENT 'api索引，保存了类和方法',
  `hash` varchar(50) DEFAULT '' COMMENT 'api唯一标识',
  `access_token` tinyint(4) DEFAULT '1' COMMENT '认证方式 1：复杂认证，0：简易认证',
  `status` tinyint(4) DEFAULT '1' COMMENT 'API状态：0表示禁用，1表示启用',
  `method` tinyint(4) DEFAULT '2' COMMENT '请求方式0：不限1：Post，2：Get',
  `info` varchar(500) DEFAULT '' COMMENT 'api中文说明',
  `is_test` tinyint(4) DEFAULT '0' COMMENT '是否是测试模式：0:生产模式，1：测试模式',
  `return_str` text COMMENT '返回数据示例',
  `group_hash` varchar(64) DEFAULT 'default' COMMENT '当前接口所属的接口分组',
  `hash_type` tinyint(4) DEFAULT '2' COMMENT '是否采用hash映射， 1：普通模式 2：加密模式',
  PRIMARY KEY (`id`),
  KEY `hash` (`hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用于维护接口信息';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_list`
--

LOCK TABLES `admin_list` WRITE;
/*!40000 ALTER TABLE `admin_list` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_list` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_menu`
--

DROP TABLE IF EXISTS `admin_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_menu` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(50) DEFAULT '' COMMENT '菜单名',
  `fid` int(11) DEFAULT '0' COMMENT '父级菜单ID',
  `url` varchar(50) DEFAULT '' COMMENT '链接',
  `auth` tinyint(4) DEFAULT '1' COMMENT '是否需要登录才可以访问，1-需要，0-不需要',
  `sort` int(11) DEFAULT '0' COMMENT '排序',
  `show` tinyint(4) DEFAULT '1' COMMENT '是否显示，1-显示，0-隐藏',
  `icon` varchar(50) DEFAULT '' COMMENT '菜单图标',
  `level` tinyint(4) DEFAULT '1' COMMENT '菜单层级，1-一级菜单，2-二级菜单，3-按钮',
  `component` varchar(255) DEFAULT '' COMMENT '前端组件',
  `router` varchar(255) DEFAULT '' COMMENT '前端路由',
  `log` tinyint(4) DEFAULT '1' COMMENT '是否记录日志，1-记录，0-不记录',
  `permission` tinyint(4) DEFAULT '1' COMMENT '是否验证权限，1-鉴权，0-放行',
  `method` tinyint(4) DEFAULT '1' COMMENT '请求方式，1-GET, 2-POST, 3-PUT, 4-DELETE',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=74 DEFAULT CHARSET=utf8mb4 COMMENT='目录信息';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_menu`
--

LOCK TABLES `admin_menu` WRITE;
/*!40000 ALTER TABLE `admin_menu` DISABLE KEYS */;
INSERT INTO `admin_menu` VALUES
(1,'用户登录',73,'admin/Login/index',0,0,0,'',2,'','',0,0,2),
(2,'用户登出',73,'admin/Login/logout',1,0,0,'',2,'','',1,0,1),
(3,'系统管理',0,'',1,1,1,'ios-build',1,'','/system',1,1,1),
(4,'菜单维护',3,'',1,1,1,'md-menu',2,'system/menu','menu',1,1,1),
(5,'菜单状态修改',4,'admin/Menu/changeStatus',1,0,1,'',3,'','',1,1,1),
(6,'新增菜单',4,'admin/Menu/add',1,0,1,'',3,'','',1,1,2),
(7,'编辑菜单',4,'admin/Menu/edit',1,0,1,'',3,'','',1,1,2),
(8,'菜单删除',4,'admin/Menu/del',1,0,1,'',3,'','',1,1,1),
(9,'用户管理',3,'',1,2,1,'ios-people',2,'system/user','user',1,1,1),
(10,'获取当前组的全部用户',9,'admin/User/getUsers',1,0,1,'',3,'','',1,1,1),
(11,'用户状态修改',9,'admin/User/changeStatus',1,0,1,'',3,'','',1,1,1),
(12,'新增用户',9,'admin/User/add',1,0,1,'',3,'','',1,1,2),
(13,'用户编辑',9,'admin/User/edit',1,0,1,'',3,'','',1,1,2),
(14,'用户删除',9,'admin/User/del',1,0,1,'',3,'','',1,1,1),
(15,'权限管理',3,'',1,3,1,'md-lock',2,'system/auth','auth',1,1,1),
(16,'权限组状态编辑',15,'admin/Auth/changeStatus',1,0,1,'',3,'','',1,1,1),
(17,'从指定组中删除指定用户',15,'admin/Auth/delMember',1,0,1,'',3,'','',1,1,1),
(18,'新增权限组',15,'admin/Auth/add',1,0,1,'',3,'','',1,1,2),
(19,'权限组编辑',15,'admin/Auth/edit',1,0,1,'',3,'','',1,1,2),
(20,'删除权限组',15,'admin/Auth/del',1,0,1,'',3,'','',1,1,1),
(21,'获取全部已开放的可选组',15,'admin/Auth/getGroups',1,0,1,'',3,'','',1,1,1),
(22,'获取组所有的权限列表',15,'admin/Auth/getRuleList',1,0,1,'',3,'','',1,1,1),
(23,'应用接入',0,'',1,2,1,'ios-appstore',1,'','/apps',1,1,1),
(24,'应用管理',23,'',1,0,1,'md-list-box',2,'app/list','appsList',1,1,1),
(25,'应用状态编辑',24,'admin/App/changeStatus',1,0,1,'',3,'','',1,1,1),
(26,'获取AppId,AppSecret,接口列表,应用接口权限细节',24,'admin/App/getAppInfo',1,0,1,'',3,'','',1,1,1),
(27,'新增应用',24,'admin/App/add',1,0,1,'',3,'','',1,1,2),
(28,'编辑应用',24,'admin/App/edit',1,0,1,'',3,'','',1,1,2),
(29,'删除应用',24,'admin/App/del',1,0,1,'',3,'','',1,1,1),
(30,'接口管理',0,'',1,3,1,'ios-link',1,'','/interface',1,1,1),
(31,'接口维护',30,'',1,0,1,'md-infinite',2,'interface/list','interfaceList',1,1,1),
(32,'接口状态编辑',31,'admin/InterfaceList/changeStatus',1,0,1,'',3,'','',1,1,1),
(33,'获取接口唯一标识',31,'admin/InterfaceList/getHash',1,0,1,'',3,'','',1,1,1),
(34,'添加接口',31,'admin/InterfaceList/add',1,0,1,'',3,'','',1,1,2),
(35,'编辑接口',31,'admin/InterfaceList/edit',1,0,1,'',3,'','',1,1,2),
(36,'删除接口',31,'admin/InterfaceList/del',1,0,1,'',3,'','',1,1,1),
(37,'获取接口请求字段',30,'admin/Fields/request',1,0,1,'',3,'interface/request','request/:hash',1,1,1),
(38,'获取接口返回字段',30,'admin/Fields/response',1,0,1,'',3,'interface/response','response/:hash',1,1,1),
(39,'添加接口字段',31,'admin/Fields/add',1,0,1,'',3,'','',1,1,2),
(40,'上传接口返回字段',31,'admin/Fields/upload',1,0,1,'',3,'','',1,1,2),
(41,'编辑接口字段',31,'admin/Fields/edit',1,0,1,'',3,'','',1,1,2),
(42,'删除接口字段',31,'admin/Fields/del',1,0,1,'',3,'','',1,1,1),
(43,'接口分组',30,'',1,1,1,'md-archive',2,'interface/group','interfaceGroup',1,1,1),
(44,'添加接口组',43,'admin/InterfaceGroup/add',1,0,1,'',3,'','',1,1,2),
(45,'编辑接口组',43,'admin/InterfaceGroup/edit',1,0,1,'',3,'','',1,1,2),
(46,'删除接口组',43,'admin/InterfaceGroup/del',1,0,1,'',3,'','',1,1,1),
(47,'获取全部有效的接口组',43,'admin/InterfaceGroup/getAll',1,0,1,'',3,'','',1,1,1),
(48,'接口组状态维护',43,'admin/InterfaceGroup/changeStatus',1,0,1,'',3,'','',1,1,1),
(49,'应用分组',23,'',1,1,1,'ios-archive',2,'app/group','appsGroup',1,1,1),
(50,'添加应用组',49,'admin/AppGroup/add',1,0,1,'',3,'','',1,1,2),
(51,'编辑应用组',49,'admin/AppGroup/edit',1,0,1,'',3,'','',1,1,2),
(52,'删除应用组',49,'admin/AppGroup/del',1,0,1,'',3,'','',1,1,1),
(53,'获取全部可用应用组',49,'admin/AppGroup/getAll',1,0,1,'',3,'','',1,1,1),
(54,'应用组状态编辑',49,'admin/AppGroup/changeStatus',1,0,1,'',3,'','',1,1,1),
(55,'菜单列表',4,'admin/Menu/index',1,0,1,'',3,'','',1,1,1),
(56,'用户列表',9,'admin/User/index',1,0,1,'',3,'','',1,1,1),
(57,'权限列表',15,'admin/Auth/index',1,0,1,'',3,'','',1,1,1),
(58,'应用列表',24,'admin/App/index',1,0,1,'',3,'','',1,1,1),
(59,'应用分组列表',49,'admin/AppGroup/index',1,0,1,'',3,'','',1,1,1),
(60,'接口列表',31,'admin/InterfaceList/index',1,0,1,'',3,'','',1,1,1),
(61,'接口分组列表',43,'admin/InterfaceGroup/index',1,0,1,'',3,'','',1,1,1),
(62,'日志管理',3,'',1,4,1,'md-clipboard',2,'system/log','log',1,1,1),
(63,'获取操作日志列表',62,'admin/Log/index',1,0,1,'',3,'','',1,1,1),
(64,'删除单条日志记录',62,'admin/Log/del',1,0,1,'',3,'','',1,1,1),
(65,'刷新路由',31,'admin/InterfaceList/refresh',1,0,1,'',3,'','',1,1,1),
(67,'文件上传',73,'admin/Index/upload',1,0,0,'',2,'','',1,1,2),
(68,'更新个人信息',73,'admin/User/own',1,0,0,'',2,'','',1,1,2),
(69,'刷新AppSecret',24,'admin/App/refreshAppSecret',1,0,1,'',3,'','',1,1,1),
(70,'获取用户信息',73,'admin/Login/getUserInfo',1,0,0,'',2,'','',0,1,1),
(71,'编辑权限细节',15,'admin/Auth/editRule',1,0,1,'',3,'','',1,1,2),
(72,'获取用户有权限的菜单',73,'admin/Login/getAccessMenu',1,0,0,'',2,'','',0,0,1),
(73,'系统支撑',0,'',0,0,0,'logo-tux',1,'','',0,0,1);
/*!40000 ALTER TABLE `admin_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_user`
--

DROP TABLE IF EXISTS `admin_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_user` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(64) DEFAULT '' COMMENT '用户名',
  `nickname` varchar(64) DEFAULT '' COMMENT '用户昵称',
  `password` char(32) DEFAULT '' COMMENT '用户密码',
  `create_time` int(11) DEFAULT '0' COMMENT '注册时间',
  `create_ip` bigint(20) DEFAULT '0' COMMENT '注册IP',
  `update_time` int(11) DEFAULT '0' COMMENT '更新时间',
  `status` tinyint(4) DEFAULT '0' COMMENT '账号状态 0封号 1正常',
  `openid` varchar(100) DEFAULT '' COMMENT '三方登录唯一ID',
  PRIMARY KEY (`id`),
  KEY `create_time` (`create_time`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COMMENT='管理员认证信息';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_user`
--

LOCK TABLES `admin_user` WRITE;
/*!40000 ALTER TABLE `admin_user` DISABLE KEYS */;
INSERT INTO `admin_user` VALUES
(1,'root','root','594786902d11be7b144ffb6c92661698',1755501717,2130706433,1755501717,1,NULL);
/*!40000 ALTER TABLE `admin_user` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_auth_rule`
--

DROP TABLE IF EXISTS `admin_auth_rule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_auth_rule` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `url` varchar(80) DEFAULT '' COMMENT '规则唯一标识',
  `group_id` int(11) unsigned DEFAULT '0' COMMENT '权限所属组的ID',
  `auth` int(11) unsigned DEFAULT '0' COMMENT '权限数值',
  `status` tinyint(4) DEFAULT '1' COMMENT '状态：为1正常，为0禁用',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='权限细节';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_auth_rule`
--

LOCK TABLES `admin_auth_rule` WRITE;
/*!40000 ALTER TABLE `admin_auth_rule` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_auth_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_fields`
--

DROP TABLE IF EXISTS `admin_fields`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_fields` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `field_name` varchar(50) DEFAULT '' COMMENT '字段名称',
  `hash` varchar(50) DEFAULT '' COMMENT '权限所属组的ID',
  `data_type` tinyint(4) DEFAULT '0' COMMENT '数据类型，来源于DataType类库',
  `default` varchar(500) DEFAULT '' COMMENT '默认值',
  `is_must` tinyint(4) DEFAULT '0' COMMENT '是否必须 0为不必须，1为必须',
  `range` varchar(500) DEFAULT '' COMMENT '范围，Json字符串，根据数据类型有不一样的含义',
  `info` varchar(500) DEFAULT '' COMMENT '字段说明',
  `type` tinyint(4) DEFAULT '0' COMMENT '字段用处：0为request，1为response',
  `show_name` varchar(50) DEFAULT '' COMMENT 'wiki显示用字段',
  PRIMARY KEY (`id`),
  KEY `hash` (`hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用于保存各个API的字段规则';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_fields`
--

LOCK TABLES `admin_fields` WRITE;
/*!40000 ALTER TABLE `admin_fields` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_fields` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_user_data`
--

DROP TABLE IF EXISTS `admin_user_data`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_user_data` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `login_times` int(11) DEFAULT '0' COMMENT '账号登录次数',
  `last_login_ip` bigint(20) DEFAULT '0' COMMENT '最后登录IP',
  `last_login_time` int(11) DEFAULT '0' COMMENT '最后登录时间',
  `uid` int(11) DEFAULT '0' COMMENT '用户ID',
  `head_img` text COMMENT '用户头像',
  PRIMARY KEY (`id`),
  KEY `uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='管理员数据表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_user_data`
--

LOCK TABLES `admin_user_data` WRITE;
/*!40000 ALTER TABLE `admin_user_data` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_user_data` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_app`
--

DROP TABLE IF EXISTS `admin_app`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_app` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `app_id` varchar(50) DEFAULT '' COMMENT '应用id',
  `app_secret` varchar(50) DEFAULT '' COMMENT '应用密码',
  `app_name` varchar(50) DEFAULT '' COMMENT '应用名称',
  `app_status` tinyint(4) DEFAULT '1' COMMENT '应用状态：0表示禁用，1表示启用',
  `app_info` text COMMENT '应用说明',
  `app_api` text COMMENT '当前应用允许请求的全部API接口',
  `app_group` varchar(128) DEFAULT 'default' COMMENT '当前应用所属的应用组唯一标识',
  `app_add_time` int(11) DEFAULT '0' COMMENT '应用创建时间',
  `app_api_show` text COMMENT '前台样式显示所需数据格式',
  PRIMARY KEY (`id`),
  UNIQUE KEY `app_id` (`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='appId和appSecret表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_app`
--

LOCK TABLES `admin_app` WRITE;
/*!40000 ALTER TABLE `admin_app` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_app` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `admin_group`
--

DROP TABLE IF EXISTS `admin_group`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_group` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(128) DEFAULT '' COMMENT '组名称',
  `description` text COMMENT '组说明',
  `status` tinyint(4) DEFAULT '1' COMMENT '状态：为1正常，为0禁用',
  `hash` varchar(128) DEFAULT '' COMMENT '组标识',
  `create_time` int(11) DEFAULT '0' COMMENT '创建时间',
  `update_time` int(11) DEFAULT '0' COMMENT '修改时间',
  `image` varchar(256) DEFAULT NULL COMMENT '分组封面图',
  `hot` int(11) DEFAULT '0' COMMENT '分组热度',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COMMENT='接口组管理';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `admin_group`
--

LOCK TABLES `admin_group` WRITE;
/*!40000 ALTER TABLE `admin_group` DISABLE KEYS */;
INSERT INTO `admin_group` VALUES
(1,'默认分组','默认分组',1,'default',1755501717,1755501717,'',0);
/*!40000 ALTER TABLE `admin_group` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*M!100616 SET NOTE_VERBOSITY=@OLD_NOTE_VERBOSITY */;

-- Dump completed on 2025-08-22 10:35:43
