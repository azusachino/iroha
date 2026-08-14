package expenses

// MaxItems is the maximum number of descriptive receipt items in an expense.
const MaxItems = 32

// SupportedCurrencies maps the accepted ISO currency code to its minor-unit exponent.
var SupportedCurrencies = map[string]int{
	"JPY": 0,
	"USD": 2,
	"EUR": 2,
	"GBP": 2,
}

// SupportedCategories contains the canonical expense categories.
var SupportedCategories = map[string]struct{}{
	"food":          {},
	"groceries":     {},
	"transport":     {},
	"shopping":      {},
	"housing":       {},
	"utilities":     {},
	"health":        {},
	"entertainment": {},
	"subscriptions": {},
	"work":          {},
	"other":         {},
}
