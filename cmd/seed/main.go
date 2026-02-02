package main

import (
	"fmt"
	"log"
	"moonshine/internal/config"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	"moonshine/internal/domain"
	"moonshine/internal/repository"
	"moonshine/internal/util"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not loaded, relying on environment")
	}

	cfg := config.Load()
	db, err := repository.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Starting seed process...")

	if err := truncateTables(db.DB()); err != nil {
		log.Fatalf("Failed to truncate tables: %v", err)
	}

	seedAvatars(db.DB())
	seedEquipmentCategories(db.DB())
	if err := seedEquipmentItems(db.DB()); err != nil {
		log.Printf("Failed to seed equipment items: %v", err)
	}
	if err := seedArtifactItems(db.DB()); err != nil {
		log.Printf("Failed to seed artifact items: %v", err)
	}
	if err := seedLocations(db.DB()); err != nil {
		log.Printf("Failed to seed locations: %v", err)
	}
	if err := seedBots(db.DB()); err != nil {
		log.Printf("Failed to seed bots: %v", err)
	}
	seedUsers(db.DB())

	log.Println("Seed process completed!")
}

func truncateTables(db *sqlx.DB) error {
	log.Println("Truncating all seed tables...")

	tables := []string{
		"inventory",
		"location_locations",
		"equipment_items",
		"equipment_categories",
		"locations",
		"avatars",
	}

	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
		log.Printf("Truncated table: %s", table)
	}

	log.Println("All tables truncated successfully")
	return nil
}

func seedAvatars(db *sqlx.DB) {
	log.Println("Seeding avatars...")

	avatarRepo := repository.NewAvatarRepository(db)

	avatarsDir := "frontend/assets/images/players/avatars"
	if _, err := os.Stat(avatarsDir); os.IsNotExist(err) {
		return
	}

	files, err := filepath.Glob(filepath.Join(avatarsDir, "*.png"))
	if err != nil {
		return
	}

	if len(files) == 0 {
		return
	}

	count := 0

	for i, file := range files {
		filename := filepath.Base(file)
		imagePath := filepath.Join("players/avatars", filename)

		_, err := avatarRepo.FindByImage(imagePath)
		if err == nil {
			continue
		}

		avatar := &domain.Avatar{
			Image:   imagePath,
			Private: false,
		}

		if err := avatarRepo.Create(avatar); err != nil {
			continue
		}

		count++
		log.Printf("Created avatar %d: %s", i+1, imagePath)
	}

	log.Printf("Successfully created %d avatars", count)
}

func seedUsers(db *sqlx.DB) {
	log.Println("Seeding users...")

	userRepo := repository.NewUserRepository(db)

	existingUser, err := userRepo.FindByUsername("admin")
	if err == nil && existingUser != nil {
		log.Println("User 'admin' already exists, skipping")
		return
	}

	hashedPassword, err := util.HashPassword("password")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	avatarRepo := repository.NewAvatarRepository(db)
	locationRepo := repository.NewLocationRepository(db)

	var firstAvatar *domain.Avatar
	avatarsDir := "frontend/assets/images/players/avatars"
	files, err := filepath.Glob(filepath.Join(avatarsDir, "*.png"))
	if err == nil && len(files) > 0 {
		filename := filepath.Base(files[0])
		imagePath := filepath.Join("players/avatars", filename)
		firstAvatar, err = avatarRepo.FindByImage(imagePath)
		if err != nil {
		}
	} else {
	}

	moonshineLocation, err := locationRepo.FindStartLocation()
	if err != nil {
		log.Fatalf("Moonshine location not found, please seed locations first: %v", err)
	}

	user := &domain.User{
		Username:   "admin",
		Name:       "admin",
		Email:      "admin@gmail.com",
		Password:   hashedPassword,
		Attack:     1,
		Defense:    1,
		Hp:         20,
		CurrentHp:  20,
		Level:      1,
		Gold:       200000,
		Exp:        0,
		FreeStats:  5,
		LocationID: moonshineLocation.ID,
	}

	if firstAvatar != nil && firstAvatar.ID != uuid.Nil {
		avatarID := firstAvatar.ID
		user.AvatarID = &avatarID
		log.Printf("Assigned avatar ID %s to user", firstAvatar.ID.String())
	}

	log.Printf("Assigned location ID %s (Moonshine) to user", moonshineLocation.ID.String())

	if err := userRepo.Create(user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	log.Printf("Successfully created user: %s (%s)", user.Username, user.Email)
}

func seedLocations(db *sqlx.DB) error {
	log.Println("Seeding locations...")

	locationRepo := repository.NewLocationRepository(db)

	moonshineLocation, err := locationRepo.FindStartLocation()
	if err == nil && moonshineLocation != nil {
		if _, err := db.Exec("UPDATE locations SET cell = false WHERE slug IN ('moonshine', 'shop_of_artifacts', 'weapon_shop')"); err != nil {
		}
		if _, err := db.Exec("UPDATE locations SET cell = true WHERE slug LIKE '%cell'"); err != nil {
		}
		return nil
	}

	moonshineLocation = &domain.Location{
		Name:     "Moonshine",
		Slug:     "moonshine",
		Cell:     false,
		Inactive: false,
		Image:    "cities/moonshine/icon.jpg",
		ImageBg:  "cities/moonshine/bg.jpg",
	}

	if err := locationRepo.Create(moonshineLocation); err != nil {
		return fmt.Errorf("failed to create Moonshine location: %w", err)
	}

	shops := []struct {
		slug string
		name string
	}{
		{"weapon_shop", "Weapon shop"},
		{"shop_of_artifacts", "Артефакты"},
	}

	shopLocations := make(map[string]uuid.UUID)

	for _, shop := range shops {
		shopName := shop.name
		shopLocation := &domain.Location{
			Name:     shopName,
			Slug:     shop.slug,
			Cell:     false,
			Inactive: false,
			Image:    fmt.Sprintf("cities/moonshine/%s/icon.png", shop.slug),
			ImageBg:  fmt.Sprintf("cities/moonshine/%s/bg.jpg", shop.slug),
		}

		if err := locationRepo.Create(shopLocation); err != nil {
			return fmt.Errorf("failed to create shop location %s: %w", shop.slug, err)
		}

		shopLocations[shop.slug] = shopLocation.ID

		locLocID := uuid.New()
		locationLocationQuery := `INSERT INTO location_locations (id, location_id, near_location_id) 
			VALUES ($1, $2, $3)`
		if _, err := db.Exec(locationLocationQuery, locLocID, moonshineLocation.ID, shopLocation.ID); err != nil {
			return fmt.Errorf("failed to create location connection for %s: %w", shop.slug, err)
		}

		locLocReverseID := uuid.New()
		if _, err := db.Exec(locationLocationQuery, locLocReverseID, shopLocation.ID, moonshineLocation.ID); err != nil {
			return fmt.Errorf("failed to create reverse location connection for %s: %w", shop.slug, err)
		}

	}

	waywardPinesLocation := &domain.Location{
		Name:     "Wayward Pines",
		Slug:     "wayward_pines",
		Cell:     false,
		Inactive: false,
		Image:    "wayward_pines/icon.png",
		ImageBg:  "wayward_pines/bg.jpg",
	}

	if err := locationRepo.Create(waywardPinesLocation); err != nil {
		return fmt.Errorf("failed to create wayward_pines location: %w", err)
	}

	waywardPinesLocLocID := uuid.New()
	locationLocationQuery := `INSERT INTO location_locations (id, location_id, near_location_id) 
		VALUES ($1, $2, $3)`
	if _, err := db.Exec(locationLocationQuery, waywardPinesLocLocID, moonshineLocation.ID, waywardPinesLocation.ID); err != nil {
		return fmt.Errorf("failed to create location connection for wayward_pines: %w", err)
	}

	waywardPinesLocLocReverseID := uuid.New()
	if _, err := db.Exec(locationLocationQuery, waywardPinesLocLocReverseID, waywardPinesLocation.ID, moonshineLocation.ID); err != nil {
		return fmt.Errorf("failed to create reverse location connection for wayward_pines: %w", err)
	}

	internalLocations := map[string]uuid.UUID{
		"moonshine":         moonshineLocation.ID,
		"shop_of_artifacts": shopLocations["shop_of_artifacts"],
		"weapon_shop":       shopLocations["weapon_shop"],
		"wayward_pines":     waywardPinesLocation.ID,
	}

	locationNames := []string{"moonshine", "shop_of_artifacts", "weapon_shop", "wayward_pines"}

	for i, loc1Name := range locationNames {
		for j, loc2Name := range locationNames {
			if i >= j {
				continue
			}

			loc1ID := internalLocations[loc1Name]
			loc2ID := internalLocations[loc2Name]

			var existingConnectionID uuid.UUID
			err := db.QueryRow(
				"SELECT id FROM location_locations WHERE location_id = $1 AND near_location_id = $2",
				loc1ID, loc2ID).Scan(&existingConnectionID)

			if err != nil {
				connID := uuid.New()
				if _, err := db.Exec(locationLocationQuery, connID, loc1ID, loc2ID); err != nil {
					return fmt.Errorf("failed to create connection %s -> %s: %w", loc1Name, loc2Name, err)
				}
			}

			var existingReverseConnectionID uuid.UUID
			err = db.QueryRow(
				"SELECT id FROM location_locations WHERE location_id = $1 AND near_location_id = $2",
				loc2ID, loc1ID).Scan(&existingReverseConnectionID)

			if err != nil {
				reverseConnID := uuid.New()
				if _, err := db.Exec(locationLocationQuery, reverseConnID, loc2ID, loc1ID); err != nil {
					return fmt.Errorf("failed to create reverse connection %s -> %s: %w", loc2Name, loc1Name, err)
				}
			}
		}
	}

	cellsDir := "frontend/assets/images/locations/wayward_pines/cells"
	files, err := filepath.Glob(filepath.Join(cellsDir, "*.png"))
	if err != nil {
		return fmt.Errorf("failed to read cells directory: %w", err)
	}

	if len(files) != 64 {
		return fmt.Errorf("expected 64 cell files, found %d", len(files))
	}

	sort.Slice(files, func(i, j int) bool {
		numI := extractCellNumber(files[i])
		numJ := extractCellNumber(files[j])
		return numI < numJ
	})

	cellLocations := make(map[int]uuid.UUID)

	for _, file := range files {
		cellNum := extractCellNumber(file)
		if cellNum == 0 {
			continue
		}

		cellSlug := fmt.Sprintf("%dcell", cellNum)
		cellLocation := &domain.Location{
			Name:     "",
			Slug:     cellSlug,
			Cell:     true,
			Inactive: false,
			Image:    fmt.Sprintf("wayward_pines/cells/%s.png", cellSlug),
			ImageBg:  "",
		}

		if err := locationRepo.Create(cellLocation); err != nil {
			return fmt.Errorf("failed to create cell location %d: %w", cellNum, err)
		}

		cellLocations[cellNum] = cellLocation.ID
	}

	for cellNum := 1; cellNum <= 64; cellNum++ {
		cellID := cellLocations[cellNum]
		row := (cellNum - 1) / 8
		col := (cellNum - 1) % 8

		neighbors := []int{}

		if col > 0 {
			neighbors = append(neighbors, cellNum-1)
		}
		if col < 7 {
			neighbors = append(neighbors, cellNum+1)
		}
		if row > 0 {
			neighbors = append(neighbors, cellNum-8)
		}
		if row < 7 {
			neighbors = append(neighbors, cellNum+8)
		}

		if row > 0 && col > 0 {
			neighbors = append(neighbors, cellNum-9)
		}
		if row > 0 && col < 7 {
			neighbors = append(neighbors, cellNum-7)
		}
		if row < 7 && col > 0 {
			neighbors = append(neighbors, cellNum+7)
		}
		if row < 7 && col < 7 {
			neighbors = append(neighbors, cellNum+9)
		}

		for _, neighborNum := range neighbors {
			neighborID := cellLocations[neighborNum]

			var existingConnectionID uuid.UUID
			err := db.QueryRow(
				"SELECT id FROM location_locations WHERE location_id = $1 AND near_location_id = $2",
				cellID, neighborID).Scan(&existingConnectionID)

			if err != nil {
				locLocID := uuid.New()
				locLocQuery := `INSERT INTO location_locations (id, location_id, near_location_id) 
					VALUES ($1, $2, $3)`
				if _, err := db.Exec(locLocQuery, locLocID, cellID, neighborID); err != nil {
					return fmt.Errorf("failed to create cell connection %d -> %d: %w", cellNum, neighborNum, err)
				}
			}

			var existingReverseConnectionID uuid.UUID
			err = db.QueryRow(
				"SELECT id FROM location_locations WHERE location_id = $1 AND near_location_id = $2",
				neighborID, cellID).Scan(&existingReverseConnectionID)

			if err != nil {
				locLocReverseID := uuid.New()
				locLocQuery := `INSERT INTO location_locations (id, location_id, near_location_id) 
					VALUES ($1, $2, $3)`
				if _, err := db.Exec(locLocQuery, locLocReverseID, neighborID, cellID); err != nil {
					return fmt.Errorf("failed to create reverse cell connection %d -> %d: %w", neighborNum, cellNum, err)
				}
			}
		}
	}

	moonshineLocation, err = locationRepo.FindBySlug("moonshine")
	if err != nil {
		return fmt.Errorf("failed to find moonshine location: %w", err)
	}

	cell29ID := cellLocations[29]
	cell37ID := cellLocations[37]

	cell29ToMoonshineID := uuid.New()
	if _, err := db.Exec(locationLocationQuery, cell29ToMoonshineID, cell29ID, moonshineLocation.ID); err != nil {
		return fmt.Errorf("failed to create connection 29cell -> moonshine: %w", err)
	}

	moonshineToCell29ID := uuid.New()
	if _, err := db.Exec(locationLocationQuery, moonshineToCell29ID, moonshineLocation.ID, cell29ID); err != nil {
		return fmt.Errorf("failed to create connection moonshine -> 29cell: %w", err)
	}

	cell37ToMoonshineID := uuid.New()
	if _, err := db.Exec(locationLocationQuery, cell37ToMoonshineID, cell37ID, moonshineLocation.ID); err != nil {
		return fmt.Errorf("failed to create connection 37cell -> moonshine: %w", err)
	}

	moonshineToCell37ID := uuid.New()
	if _, err := db.Exec(locationLocationQuery, moonshineToCell37ID, moonshineLocation.ID, cell37ID); err != nil {
		return fmt.Errorf("failed to create connection moonshine -> 37cell: %w", err)
	}

	return nil
}

func seedBots(db *sqlx.DB) error {
	log.Println("Seeding bots...")

	botRepo := repository.NewBotRepository(db)
	locationRepo := repository.NewLocationRepository(db)

	var existingBotID uuid.UUID
	err := db.QueryRow("SELECT id FROM bots WHERE slug = $1 AND deleted_at IS NULL", "rat").Scan(&existingBotID)
	if err != nil {
		ratBot := &domain.Bot{
			Name:    "Крыса",
			Slug:    "rat",
			Attack:  2,
			Defense: 10,
			Hp:      20,
			Level:   1,
			Avatar:  "images/bots/rat.jpg",
		}

		if err := botRepo.Create(ratBot); err != nil {
			return fmt.Errorf("failed to create rat bot: %w", err)
		}

		log.Printf("Created bot: Крыса (ID: %s)", ratBot.ID.String())
		existingBotID = ratBot.ID
	} else {
		log.Println("Bot 'rat' already exists")
	}

	cell29Location, err := locationRepo.FindBySlug("29cell")
	if err != nil {
		return fmt.Errorf("failed to find 29cell location: %w", err)
	}

	var existingLinkID uuid.UUID
	err = db.QueryRow(
		"SELECT id FROM location_bots WHERE location_id = $1 AND bot_id = $2 AND deleted_at IS NULL",
		cell29Location.ID, existingBotID,
	).Scan(&existingLinkID)

	if err != nil {
		linkID := uuid.New()
		linkQuery := `INSERT INTO location_bots (id, location_id, bot_id) VALUES ($1, $2, $3)`
		if _, err := db.Exec(linkQuery, linkID, cell29Location.ID, existingBotID); err != nil {
			return fmt.Errorf("failed to link rat bot to 29cell: %w", err)
		}
		log.Printf("Linked bot 'Крыса' to location 29cell")
	} else {
		log.Println("Bot 'rat' already linked to 29cell")
	}

	log.Println("Bots seeding completed!")
	return nil
}

func seedEquipmentCategories(db *sqlx.DB) {
	log.Println("Seeding equipment categories...")

	categories := []struct {
		name string
		typ  string
	}{
		{"Chest", "chest"},
		{"Belt", "belt"},
		{"Head", "head"},
		{"Neck", "neck"},
		{"Weapon", "weapon"},
		{"Shield", "shield"},
		{"Legs", "legs"},
		{"Feet", "feet"},
		{"Arms", "arms"},
		{"Hands", "hands"},
		{"Ring", "ring"},
	}

	for _, cat := range categories {
		var existingID uuid.UUID
		err := db.QueryRow("SELECT id FROM equipment_categories WHERE type = $1", cat.typ).Scan(&existingID)
		if err == nil {
			log.Printf("Equipment category %s already exists, skipping", cat.name)
			continue
		}

		categoryID := uuid.New()
		query := `INSERT INTO equipment_categories (id, name, type) VALUES ($1, $2, $3)`
		if _, err := db.Exec(query, categoryID, cat.name, cat.typ); err != nil {
			log.Printf("Failed to create equipment category %s: %v", cat.name, err)
			continue
		}
		log.Printf("Created equipment category: %s (%s)", cat.name, cat.typ)
	}

	log.Println("Equipment categories seeding completed!")
}

type equipmentFileInfo struct {
	path          string
	categoryType  string
	name          string
	price         uint
	attack        uint
	requiredLevel uint
	hp            uint
	defense       uint
}

func seedEquipmentItems(db *sqlx.DB) error {
	log.Println("Seeding equipment items...")

	baseDir := "frontend/assets/images/equipment_items"

	categoryMap := map[string]string{
		"chest":  "chest",
		"belt":   "belt",
		"head":   "head",
		"neck":   "neck",
		"weapon": "weapon",
		"shield": "shield",
		"legs":   "legs",
		"feet":   "feet",
		"arms":   "arms",
		"hands":  "hands",
		"ring":   "ring",
	}

	var allFiles []equipmentFileInfo

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".png" {
			return nil
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		if len(parts) < 1 {
			return nil
		}

		dir := parts[0]
		categoryType := categoryMap[dir]

		if categoryType == "" {
			return nil
		}

		fileName := filepath.Base(path)
		fileInfo := equipmentFileInfo{
			path:         path,
			categoryType: categoryType,
		}

		if !parseEquipmentFileName(fileName, &fileInfo) {
			return nil
		}

		allFiles = append(allFiles, fileInfo)
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk equipment items directory: %w", err)
	}

	categoryIDs := make(map[string]uuid.UUID)
	for catType := range categoryMap {
		var catID uuid.UUID
		err := db.QueryRow("SELECT id FROM equipment_categories WHERE type = $1", catType).Scan(&catID)
		if err == nil {
			categoryIDs[catType] = catID
		}
	}
	var weaponCatID uuid.UUID
	err = db.QueryRow("SELECT id FROM equipment_categories WHERE type = 'weapon'").Scan(&weaponCatID)
	if err == nil {
		categoryIDs["weapon"] = weaponCatID
	}

	count := 0
	for _, file := range allFiles {
		catID := categoryIDs[file.categoryType]
		if catID == uuid.Nil {
			continue
		}

		dbImagePath := strings.TrimPrefix(file.path, "frontend/assets/images/")
		dbImagePath = strings.ReplaceAll(dbImagePath, "\\", "/")

		var existingID uuid.UUID
		err := db.QueryRow("SELECT id FROM equipment_items WHERE image = $1", dbImagePath).Scan(&existingID)
		if err == nil {
			continue
		}

		slug := generateSlugFromImage(dbImagePath)

		itemID := uuid.New()
		query := `INSERT INTO equipment_items 
			(id, name, slug, attack, defense, hp, required_level, price, artifact, equipment_category_id, image) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

		_, err = db.Exec(query,
			itemID,
			file.name,
			slug,
			file.attack,
			file.defense,
			file.hp,
			file.requiredLevel,
			file.price,
			false,
			catID,
			dbImagePath,
		)

		if err != nil {
			log.Printf("Failed to create equipment item %s: %v", file.name, err)
			continue
		}

		count++
		log.Printf("Created equipment item: %s (level %d, attack %d, defense %d, hp %d, price %d)",
			file.name, file.requiredLevel, file.attack, file.defense, file.hp, file.price)
	}

	log.Printf("Equipment items seeding completed! Created %d items", count)
	return nil
}

var artifactCategoryMap = map[string]string{
	"arms": "arms", "belt": "belt", "chest": "chest", "feet": "feet",
	"hands": "hands", "head": "head", "helm": "head", "legs": "legs",
	"neck": "neck", "ring": "ring", "shield": "shield", "weapon": "weapon",
}

type artifactFileInfo struct {
	path          string
	categoryType  string
	name          string
	price         uint
	attack        uint
	requiredLevel uint
	hp            uint
	defense       uint
}

func parseArtifactFileName(filename string, info *artifactFileInfo) bool {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(base, "-")
	if len(parts) < 7 {
		return false
	}
	for i := len(parts) - 5; i < len(parts); i++ {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return false
		}
	}
	catKey := strings.ToLower(parts[0])
	info.categoryType = artifactCategoryMap[catKey]
	if info.categoryType == "" {
		return false
	}
	nameParts := parts[1 : len(parts)-5]
	info.name = strings.Join(nameParts, " ")
	info.name = strings.ReplaceAll(info.name, "_", " ")
	info.name = strings.ReplaceAll(info.name, "-", " ")
	if info.name == "" {
		return false
	}
	runes := []rune(info.name)
	if len(runes) > 0 {
		r := runes[0]
		if r >= 'а' && r <= 'я' {
			runes[0] = 'А' + (r - 'а')
		} else if r >= 'a' && r <= 'z' {
			runes[0] = 'A' + (r - 'a')
		}
		info.name = string(runes)
	}
	n := len(parts)
	if p, e := strconv.ParseUint(parts[n-5], 10, 32); e == nil {
		info.price = uint(p)
	} else {
		return false
	}
	if a, e := strconv.ParseUint(parts[n-4], 10, 32); e == nil {
		info.attack = uint(a)
	} else {
		return false
	}
	if l, e := strconv.ParseUint(parts[n-3], 10, 32); e == nil {
		info.requiredLevel = uint(l)
	} else {
		return false
	}
	if h, e := strconv.ParseUint(parts[n-2], 10, 32); e == nil {
		info.hp = uint(h)
	} else {
		return false
	}
	if d, e := strconv.ParseUint(parts[n-1], 10, 32); e == nil {
		info.defense = uint(d)
	} else {
		return false
	}
	return true
}

func seedArtifactItems(db *sqlx.DB) error {
	log.Println("Seeding artifact items...")

	baseDir := "frontend/assets/images/artifacts"
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}

	categoryIDs := make(map[string]uuid.UUID)
	for _, catType := range []string{"arms", "belt", "chest", "feet", "hands", "head", "legs", "neck", "ring", "shield", "weapon"} {
		var catID uuid.UUID
		if err := db.QueryRow("SELECT id FROM equipment_categories WHERE type = $1", catType).Scan(&catID); err != nil {
			continue
		}
		categoryIDs[catType] = catID
	}

	var count int
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.ToLower(filepath.Ext(path)) != ".png" {
			return nil
		}
		fileName := filepath.Base(path)
		var af artifactFileInfo
		af.path = path
		if !parseArtifactFileName(fileName, &af) {
			return nil
		}
		catID := categoryIDs[af.categoryType]
		if catID == uuid.Nil {
			return nil
		}
		dbImagePath := strings.TrimPrefix(path, "frontend/assets/images")
		dbImagePath = strings.TrimPrefix(dbImagePath, "/")
		dbImagePath = strings.TrimPrefix(dbImagePath, "\\")
		dbImagePath = filepath.ToSlash(dbImagePath)
		var exist uuid.UUID
		if db.QueryRow("SELECT id FROM equipment_items WHERE image = $1", dbImagePath).Scan(&exist) == nil {
			return nil
		}
		slug := generateSlugFromImage(dbImagePath)
		itemID := uuid.New()
		q := `INSERT INTO equipment_items (id, name, slug, attack, defense, hp, required_level, price, artifact, equipment_category_id, image)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
		_, execErr := db.Exec(q, itemID, af.name, slug, af.attack, af.defense, af.hp, af.requiredLevel, af.price, true, catID, dbImagePath)
		if execErr != nil {
			log.Printf("Failed to create artifact %s: %v", af.name, execErr)
			return nil
		}
		count++
		log.Printf("Created artifact: %s (%s)", af.name, af.categoryType)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk artifacts: %w", err)
	}
	log.Printf("Artifact items seeding completed! Created %d items", count)
	return nil
}

func parseEquipmentFileName(filename string, info *equipmentFileInfo) bool {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	parts := strings.Split(base, "-")
	if len(parts) < 7 {
		return false
	}

	nameParts := []string{}
	numStartIdx := -1
	for i := 1; i < len(parts); i++ {
		if _, err := strconv.Atoi(parts[i]); err == nil {
			if i+1 < len(parts) {
				if _, err2 := strconv.Atoi(parts[i+1]); err2 == nil {
					numStartIdx = i
					break
				}
			}
		}
		nameParts = append(nameParts, parts[i])
	}

	if numStartIdx == -1 || len(nameParts) == 0 {
		return false
	}

	info.name = strings.Join(nameParts, " ")
	info.name = strings.ReplaceAll(info.name, "_", " ")
	info.name = strings.ReplaceAll(info.name, "-", " ")

	if len(info.name) > 0 {
		runes := []rune(info.name)
		firstRune := runes[0]
		if firstRune >= 'а' && firstRune <= 'я' {
			runes[0] = 'А' + (firstRune - 'а')
		} else if firstRune >= 'a' && firstRune <= 'z' {
			runes[0] = 'A' + (firstRune - 'a')
		}
		info.name = string(runes)
	}

	if numStartIdx+4 < len(parts) {
		if price, err := strconv.ParseUint(parts[numStartIdx], 10, 32); err == nil {
			info.price = uint(price)
		} else {
			return false
		}
		if attack, err := strconv.ParseUint(parts[numStartIdx+1], 10, 32); err == nil {
			info.attack = uint(attack)
		} else {
			return false
		}
		if level, err := strconv.ParseUint(parts[numStartIdx+2], 10, 32); err == nil {
			info.requiredLevel = uint(level)
		} else {
			return false
		}
		if hp, err := strconv.ParseUint(parts[numStartIdx+3], 10, 32); err == nil {
			info.hp = uint(hp)
		} else {
			return false
		}
		defenseStr := parts[numStartIdx+4]
		defenseStr = strings.TrimSuffix(defenseStr, ".png")
		if defense, err := strconv.ParseUint(defenseStr, 10, 32); err == nil {
			info.defense = uint(defense)
		} else {
			return false
		}
		return true
	}

	return false
}

func extractCellNumber(filename string) int {
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, ".png")
	base = strings.TrimSuffix(base, "cell")

	num, err := strconv.Atoi(base)
	if err != nil {
		return 0
	}

	return num
}

func generateSlugFromImage(imagePath string) string {
	hash1 := uint32(0)
	hash2 := uint32(0)
	for i, r := range imagePath {
		hash1 = hash1*31 + uint32(r)
		hash2 = hash2*37 + uint32(r)*uint32(i+1)
	}

	combinedHash := hash1 ^ hash2

	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 6)
	for i := 0; i < 6; i++ {
		result[i] = chars[combinedHash%uint32(len(chars))]
		combinedHash /= uint32(len(chars))
		if combinedHash == 0 {
			combinedHash = hash1 + hash2
		}
	}

	return string(result)
}
