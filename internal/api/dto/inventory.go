package dto

type InventoryResponse struct {
	EquipmentItems []*EquipmentItem `json:"equipmentItems"`
	Potions        []*Potion        `json:"potions"`
	ToolItems      []*ToolItem      `json:"toolItems"`
	Resources      []*Resource      `json:"resources"`
}
