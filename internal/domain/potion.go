package domain

type Potion struct {
	Model
	Name  string `db:"name"`
	Slug  string `db:"slug"`
	Stat  string `db:"stat"`
	Value uint   `db:"value"`
	Price uint   `db:"price"`
	Image string `db:"image"`
}
