package router

import (
	"exchange_cdk/config"
	"exchange_cdk/handler"
	"exchange_cdk/middleware"
	"exchange_cdk/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Services struct {
	Auth         *service.AuthService
	User         *service.UserService
	Cdk          *service.CdkService
	Exchange     *service.ExchangeService
	Redeem       *service.RedeemService
	RedeemItem   *service.RedeemItemService
	Category     *service.RedeemCategoryService
	Template     *service.TemplateService
	Team         *service.TeamService
	Announcement *service.AnnouncementService
	Stats        *service.StatsService
}

// Setup 注册所有路由
func Setup(r *gin.Engine, svc *Services, cfg *config.Config, logger *zap.Logger, db *gorm.DB) {
	// 健康检查(无需认证)
	r.GET("/healthz", handler.Healthz)
	r.GET("/readyz", handler.Readyz(db))

	api := r.Group("/api")

	// 认证路由
	authH := handler.NewAuthHandler(svc.Auth)
	api.POST("/auth/login", authH.Login)

	redeemH := handler.NewRedeemHandler(svc.Redeem)
	api.POST("/redeem", redeemH.Redeem)

	// 需登录路由
	auth := api.Group("")
	auth.Use(middleware.AuthRequired(cfg.JWTSecret))

	userH := handler.NewUserHandler(svc.User, svc.Exchange)
	auth.GET("/user/me", userH.Me)
	auth.PUT("/user/password", userH.ChangePassword)
	auth.GET("/user/orders", userH.Orders)
	auth.GET("/user/orders/:id", userH.OrderDetail)

	exchangeH := handler.NewExchangeHandler(svc.Exchange)
	auth.GET("/amounts", exchangeH.Amounts)
	exchange := auth.Group("")
	exchange.Use(middleware.ExchangeRateLimit())
	exchange.POST("/exchange", exchangeH.Exchange)

	announceH := handler.NewAnnouncementHandler(svc.Announcement)
	auth.GET("/announcements", announceH.ListPublic)

	// 后台资源路由: 登录用户可管理自己的数据,团队共享数据只读
	admin := auth.Group("/admin")

	adminCdkH := handler.NewAdminCdkHandler(svc.Cdk)
	admin.GET("/cdk/list", adminCdkH.List)

	adminRedeemItemH := handler.NewAdminRedeemItemHandler(svc.RedeemItem)
	admin.GET("/redeem-items", adminRedeemItemH.List)
	admin.POST("/redeem-items", adminRedeemItemH.Create)
	admin.POST("/redeem-items/import", adminRedeemItemH.ImportFiles)
	admin.PUT("/redeem-items/:id", adminRedeemItemH.Update)
	admin.DELETE("/redeem-items/:id", adminRedeemItemH.Delete)

	adminRedeemCategoryH := handler.NewAdminRedeemCategoryHandler(svc.Category)
	admin.GET("/redeem-categories", adminRedeemCategoryH.List)
	admin.POST("/redeem-categories", adminRedeemCategoryH.Create)
	admin.PUT("/redeem-categories/:id", adminRedeemCategoryH.Update)
	admin.DELETE("/redeem-categories/:id", adminRedeemCategoryH.Delete)

	adminTemplateH := handler.NewAdminTemplateHandler(svc.Template)
	admin.GET("/templates", adminTemplateH.List)
	admin.POST("/templates", adminTemplateH.Create)
	admin.PUT("/templates/:id", adminTemplateH.Update)
	admin.DELETE("/templates/:id", adminTemplateH.Delete)

	adminTeamH := handler.NewAdminTeamHandler(svc.Team)
	admin.GET("/teams/my", adminTeamH.MyTeam)
	admin.GET("/teams/joined", adminTeamH.JoinedTeams)
	admin.POST("/teams/join", adminTeamH.Join)
	admin.DELETE("/teams/members/:member_id", adminTeamH.RemoveMember)

	// 全局管理路由: 仅管理员可用
	adminOnly := admin.Group("")
	adminOnly.Use(middleware.AdminRequired())

	adminUserH := handler.NewAdminUserHandler(svc.User)
	adminOnly.GET("/users", adminUserH.List)
	adminOnly.POST("/users", adminUserH.Create)
	adminOnly.PATCH("/users/:id", adminUserH.Update)

	adminOnly.POST("/announcements", announceH.Create)
	adminOnly.PUT("/announcements/:id", announceH.Update)
	adminOnly.DELETE("/announcements/:id", announceH.Delete)

	statsH := handler.NewStatsHandler(svc.Stats)
	adminOnly.GET("/stats/overview", statsH.Overview)
	adminOnly.GET("/stats/by-amount", statsH.ByAmount)
	adminOnly.GET("/stats/by-item", statsH.ByItem)
	adminOnly.GET("/stats/daily", statsH.Daily)
	adminOnly.GET("/stats/top-users", statsH.TopUsers)
	adminOnly.GET("/stats/imports", statsH.Imports)
	adminOnly.GET("/stats/by-user-amount", statsH.ByUserAmount)
}
