package leekwars

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Effect est un effet d'arme ou de puce tel que renvoyé par l'API.
type Effect struct {
	ID        int     `json:"id"`
	Value1    float64 `json:"value1"`
	Value2    float64 `json:"value2"`
	Turns     int     `json:"turns"`
	Targets   int     `json:"targets"`
	Modifiers int     `json:"modifiers"`
	Type      int     `json:"type"`
}

// Weapon porte deux identifiants : ID (celui des actions SET_WEAPON du rapport)
// et Item (celui de leek/get → weapons[].template).
type Weapon struct {
	ID         int      `json:"id"`
	Item       int      `json:"item"`
	Name       string   `json:"name"`
	Level      int      `json:"level"`
	MinRange   int      `json:"min_range"`
	MaxRange   int      `json:"max_range"`
	LaunchType int      `json:"launch_type"`
	Cost       int      `json:"cost"`
	Area       int      `json:"area"`
	Los        bool     `json:"los"`
	MaxUses    int      `json:"max_uses"`
	Effects    []Effect `json:"effects"`
}

// Chip porte aussi deux identifiants : ID (leek/get → chips[].template,
// inventaire de l'éleveur, loadouts et owned_chips) et Template (actions
// USE_CHIP des rapports de combat). Les deux espaces se recouvrent sans
// coïncider : id 14 = leather_boots, template 14 = rockfall.
type Chip struct {
	ID         int      `json:"id"`
	Template   int      `json:"template"`
	Name       string   `json:"name"`
	Level      int      `json:"level"`
	MinRange   int      `json:"min_range"`
	MaxRange   int      `json:"max_range"`
	LaunchType int      `json:"launch_type"`
	Cost       int      `json:"cost"`
	Area       int      `json:"area"`
	Los        bool     `json:"los"`
	Cooldown   int      `json:"cooldown"`
	MaxUses    int      `json:"max_uses"`
	Effects    []Effect `json:"effects"`
}

// Match est un résultat de recherche : exactement un des deux pointeurs est non nul.
type Match struct {
	Kind   string
	Weapon *Weapon
	Chip   *Chip
}

// Items est le cache des armes et puces du jeu.
type Items struct {
	weaponsByID     map[int]*Weapon
	weaponsByItem   map[int]*Weapon
	chipsByID       map[int]*Chip
	chipsByTemplate map[int]*Chip
	weapons         []*Weapon
	chips           []*Chip
}

// ParseItems construit le cache à partir des corps de weapon/get-all et chip/get-all.
func ParseItems(weaponsJSON, chipsJSON []byte) (*Items, error) {
	var w struct {
		Weapons map[string]*Weapon `json:"weapons"`
	}
	if err := json.Unmarshal(weaponsJSON, &w); err != nil {
		return nil, fmt.Errorf("weapon/get-all : %w", err)
	}
	var c struct {
		Chips map[string]*Chip `json:"chips"`
	}
	if err := json.Unmarshal(chipsJSON, &c); err != nil {
		return nil, fmt.Errorf("chip/get-all : %w", err)
	}
	items := &Items{
		weaponsByID:     map[int]*Weapon{},
		weaponsByItem:   map[int]*Weapon{},
		chipsByID:       map[int]*Chip{},
		chipsByTemplate: map[int]*Chip{},
	}
	for _, wp := range w.Weapons {
		items.weaponsByID[wp.ID] = wp
		items.weaponsByItem[wp.Item] = wp
		items.weapons = append(items.weapons, wp)
	}
	for _, ch := range c.Chips {
		items.chipsByID[ch.ID] = ch
		items.chipsByTemplate[ch.Template] = ch
		items.chips = append(items.chips, ch)
	}
	return items, nil
}

// WeaponByID résout l'identifiant utilisé dans les rapports de combat.
func (it *Items) WeaponByID(id int) *Weapon { return it.weaponsByID[id] }

// WeaponByItem résout l'identifiant utilisé dans leek/get (template).
func (it *Items) WeaponByItem(item int) *Weapon { return it.weaponsByItem[item] }

// ChipByID résout l'identifiant utilisé dans leek/get, l'inventaire et les loadouts.
func (it *Items) ChipByID(id int) *Chip { return it.chipsByID[id] }

// ChipByTemplate résout l'identifiant utilisé par les actions USE_CHIP des rapports.
func (it *Items) ChipByTemplate(template int) *Chip { return it.chipsByTemplate[template] }

// WeaponName renvoie le nom d'une arme par ID de rapport, ou "weapon_<id>".
func (it *Items) WeaponName(id int) string {
	if w := it.WeaponByID(id); w != nil {
		return w.Name
	}
	return "weapon_" + strconv.Itoa(id)
}

// ChipName renvoie le nom d'une puce par id (leek/get, inventaire, loadouts), ou "chip_<id>".
func (it *Items) ChipName(id int) string {
	if c := it.ChipByID(id); c != nil {
		return c.Name
	}
	return "chip_" + strconv.Itoa(id)
}

// ChipNameByTemplate renvoie le nom d'une puce par template (rapports), ou "chip_<template>".
func (it *Items) ChipNameByTemplate(template int) string {
	if c := it.ChipByTemplate(template); c != nil {
		return c.Name
	}
	return "chip_" + strconv.Itoa(template)
}

// Find cherche par identifiant numérique (arme : id ou item ; puce : id ou template) ou
// par nom, sans tenir compte de la casse ni des séparateurs (« sun spear » = sun_spear).
func (it *Items) Find(query string) []Match {
	var out []Match
	q := strings.TrimSpace(query)
	if n, err := strconv.Atoi(q); err == nil {
		seen := map[*Weapon]bool{}
		for _, w := range []*Weapon{it.WeaponByID(n), it.WeaponByItem(n)} {
			if w != nil && !seen[w] {
				seen[w] = true
				out = append(out, Match{Kind: "weapon", Weapon: w})
			}
		}
		seenChip := map[*Chip]bool{}
		for _, c := range []*Chip{it.ChipByID(n), it.ChipByTemplate(n)} {
			if c != nil && !seenChip[c] {
				seenChip[c] = true
				out = append(out, Match{Kind: "chip", Chip: c})
			}
		}
		return out
	}
	key := normalize(q)
	for _, w := range it.weapons {
		if normalize(w.Name) == key {
			out = append(out, Match{Kind: "weapon", Weapon: w})
		}
	}
	for _, c := range it.chips {
		if normalize(c.Name) == key {
			out = append(out, Match{Kind: "chip", Chip: c})
		}
	}
	return out
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer(" ", "", "_", "", "-", "").Replace(s)
	return s
}

// ItemsLoader charge le cache une seule fois via le client ; un échec réseau
// n'est pas mémorisé, l'appel suivant réessaie.
type ItemsLoader struct {
	client *Client
	mu     sync.Mutex
	items  *Items
}

// NewItemsLoader prépare un chargement différé.
func NewItemsLoader(c *Client) *ItemsLoader { return &ItemsLoader{client: c} }

// Items renvoie le cache, chargé au premier appel réussi (deux requêtes publiques).
func (l *ItemsLoader) Items(ctx context.Context) (*Items, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.items != nil {
		return l.items, nil
	}
	weapons, err := l.client.Get(ctx, "weapon/get-all")
	if err != nil {
		return nil, err
	}
	chips, err := l.client.Get(ctx, "chip/get-all")
	if err != nil {
		return nil, err
	}
	items, err := ParseItems(weapons, chips)
	if err != nil {
		return nil, err
	}
	l.items = items
	return items, nil
}
