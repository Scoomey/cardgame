package main

type Card struct {
	Name  string            `json:"name"`
	Stats map[string]int    `json:"stats"`
	Image string            `json:"image"`
}

var Deck = []Card{
	{
		Name: "Dragon",
		Stats: map[string]int{
			"Strength": 95,
			"Speed":    60,
			"Magic":    90,
		},
		Image: "https://example.com/images/dragon.png",
	},
	{
		Name: "Phoenix",
		Stats: map[string]int{
			"Strength": 70,
			"Speed":    80,
			"Magic":    95,
		},
		Image: "https://example.com/images/phoenix.png",
	},
	{
		Name: "Unicorn",
		Stats: map[string]int{
			"Strength": 65,
			"Speed":    75,
			"Magic":    85,
		},
		Image: "https://example.com/images/unicorn.png",
	},
	{
		Name: "Kraken",
		Stats: map[string]int{
			"Strength": 90,
			"Speed":    40,
			"Magic":    70,
		},
		Image: "https://example.com/images/kraken.png",
	},
	{
		Name: "Pegasus",
		Stats: map[string]int{
			"Strength": 60,
			"Speed":    95,
			"Magic":    75,
		},
		Image: "https://example.com/images/pegasus.png",
	},
	{
		Name: "Basilisk",
		Stats: map[string]int{
			"Strength": 80,
			"Speed":    70,
			"Magic":    85,
		},
		Image: "https://example.com/images/basilisk.png",
	},
	{
		Name: "Griffin",
		Stats: map[string]int{
			"Strength": 85,
			"Speed":    80,
			"Magic":    60,
		},
		Image: "https://example.com/images/griffin.png",
	},
	{
		Name: "White Tiger",
		Stats: map[string]int{
			"Strength": 75,
			"Speed":    85,
			"Magic":    50,
		},
		Image: "https://example.com/images/white_tiger.png",
	},
}
