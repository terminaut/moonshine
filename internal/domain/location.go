package domain

type Location struct {
	Model
	Name     string `db:"name"`
	Slug     string `db:"slug"`
	Cell     bool   `db:"cell"`
	Inactive bool   `db:"inactive"`
	Image    string `db:"image"`
	ImageBg  string `db:"image_bg"`
}

type LocationCell struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Image    string `json:"image"`
	Inactive bool   `json:"inactive"`
}

const (
	WaywardPinesSlug = "wayward_pines"
	MoonshineSlug    = "moonshine"
)
