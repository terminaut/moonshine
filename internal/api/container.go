package api

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"moonshine/internal/api/services"
	"moonshine/internal/api/ws"
	"moonshine/internal/config"
	"moonshine/internal/domain"
	r "moonshine/internal/redis"
	"moonshine/internal/repository"
	"moonshine/internal/worker"
)

type Container struct {
	UserRepo          *repository.UserRepository
	AvatarRepo        *repository.AvatarRepository
	LocationRepo      *repository.LocationRepository
	EquipmentItemRepo *repository.EquipmentItemRepository
	InventoryRepo     *repository.InventoryRepository
	PotionRepo        *repository.PotionRepository
	BotRepo           *repository.BotRepository
	FightRepo         *repository.FightRepository
	RoundRepo         *repository.RoundRepository

	UserCache  r.Cache[domain.User]
	CellsCache r.Cache[[]domain.LocationCell]

	AuthService                 *services.AuthService
	UserService                 *services.UserService
	AvatarService               *services.AvatarService
	LocationService             *services.LocationService
	EquipmentItemService        *services.EquipmentItemService
	EquipmentItemBuyService     *services.EquipmentItemBuyService
	EquipmentItemSellService    *services.EquipmentItemSellService
	EquipmentItemTakeOnService  *services.EquipmentItemTakeOnService
	EquipmentItemTakeOffService *services.EquipmentItemTakeOffService
	PotionBuyService            *services.PotionBuyService
	PotionSellService           *services.PotionSellService
	PotionTakeOnService         *services.PotionTakeOnService
	PotionTakeOffService        *services.PotionTakeOffService
	PotionUseService            *services.PotionUseService
	InventoryService            *services.InventoryService
	BotService                  *services.BotService
	FightService                *services.FightService
	HealthRegenerationService   *services.HealthRegenerationService

	Hub    *ws.Hub
	Config *config.Config
	DB     *sqlx.DB
}

func NewContainer(db *sqlx.DB, rdb *redis.Client, cfg *config.Config) (*Container, error) {
	userRepo := repository.NewUserRepository(db)
	avatarRepo := repository.NewAvatarRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	equipmentItemRepo := repository.NewEquipmentItemRepository(db)
	inventoryRepo := repository.NewInventoryRepository(db)
	potionRepo := repository.NewPotionRepository(db)
	botRepo := repository.NewBotRepository(db)
	fightRepo := repository.NewFightRepository(db)
	roundRepo := repository.NewRoundRepository(db)

	userCache := r.NewJSONCache[domain.User](rdb, "user", 5*time.Second)
	cellsCache := r.NewJSONCache[[]domain.LocationCell](rdb, "location_cells", 10*time.Minute)

	movingWorker := worker.NewCellsMovingWorker(locationRepo, userRepo, userCache, 5*time.Second)

	locationService, err := services.NewLocationService(db, locationRepo, userRepo, movingWorker, userCache, cellsCache)
	if err != nil {
		return nil, err
	}

	return &Container{
		UserRepo:          userRepo,
		AvatarRepo:        avatarRepo,
		LocationRepo:      locationRepo,
		EquipmentItemRepo: equipmentItemRepo,
		InventoryRepo:     inventoryRepo,
		PotionRepo:        potionRepo,
		BotRepo:           botRepo,
		FightRepo:         fightRepo,
		RoundRepo:         roundRepo,

		UserCache:  userCache,
		CellsCache: cellsCache,

		AuthService:                 services.NewAuthService(userRepo, avatarRepo, locationRepo, cfg.JWTKey),
		UserService:                 services.NewUserService(userRepo, avatarRepo, locationRepo, userCache),
		AvatarService:               services.NewAvatarService(avatarRepo),
		LocationService:             locationService,
		EquipmentItemService:        services.NewEquipmentItemService(equipmentItemRepo),
		EquipmentItemBuyService:     services.NewEquipmentItemBuyService(db, equipmentItemRepo, inventoryRepo, userRepo),
		EquipmentItemSellService:    services.NewEquipmentItemSellService(db, equipmentItemRepo, inventoryRepo, userRepo),
		EquipmentItemTakeOnService:  services.NewEquipmentItemTakeOnService(db, equipmentItemRepo, inventoryRepo, userRepo),
		EquipmentItemTakeOffService: services.NewEquipmentItemTakeOffService(db, equipmentItemRepo, inventoryRepo, userRepo),
		PotionBuyService:            services.NewPotionBuyService(db, potionRepo, inventoryRepo, userRepo),
		PotionSellService:           services.NewPotionSellService(db, potionRepo, userRepo),
		PotionTakeOnService:         services.NewPotionTakeOnService(db, potionRepo, inventoryRepo, userRepo),
		PotionTakeOffService:        services.NewPotionTakeOffService(db, potionRepo, inventoryRepo, userRepo),
		PotionUseService:            services.NewPotionUseService(db, potionRepo, userRepo),
		InventoryService:            services.NewInventoryService(inventoryRepo),
		BotService:                  services.NewBotService(locationRepo, botRepo, userRepo, fightRepo, roundRepo),
		FightService:                services.NewFightService(db, fightRepo, botRepo, userRepo, roundRepo),
		HealthRegenerationService:   services.NewHealthRegenerationService(userRepo),

		Hub:    ws.NewHub(),
		Config: cfg,
		DB:     db,
	}, nil
}
