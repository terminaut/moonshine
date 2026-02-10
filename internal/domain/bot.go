package domain

type Bot struct {
	Model
	Name    string `db:"name"`
	Slug    string `db:"slug"`
	Avatar  string `db:"avatar"`
	Attack  uint   `db:"attack"`
	Defense uint   `db:"defense"`
	Hp      uint   `db:"hp"`
	Level   uint   `db:"level"`
}
