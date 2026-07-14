package models

// OverviewResponse is the full response object for the dashboard overview
type OverviewResponse struct {
    Stats               Stats                   `json:"stats"`
    BooksPerMonth       []BooksPerMonth         `json:"booksPerMonth"`
    CategoryData        []CategoryData          `json:"categoryData"`
    TopBorrowedBooks    []TopBorrowedBook       `json:"topBorrowedBooks"`
    SubscriptionPlans   []SubscriptionPlanCount `json:"subscriptionPlans"`
    SubscriptionHistory []SubscriptionHistory   `json:"subscriptionHistory"`
}

// Stats represents the top-level metrics
type Stats struct {
    TotalBooks         int     `json:"totalBooks"`
    ActiveUsers        int     `json:"activeUsers"`
    TotalSubscriptions int     `json:"totalSubscriptions"`
    RevenueMonth       float64 `json:"revenueMonth"`
}

// BooksPerMonth represents books added per month
type BooksPerMonth struct {
    Month string `json:"month"`
    Books int    `json:"books"`
}

// CategoryData represents the genre/category breakdown
type CategoryData struct {
    Name  string `json:"name"`
    Value int    `json:"value"`
}

// TopBorrowedBook represents the most borrowed books
type TopBorrowedBook struct {
    Title string `json:"title"`
    Count int    `json:"count"`
}

// SubscriptionPlanCount represents active subscription counts per plan
type SubscriptionPlanCount struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}

// SubscriptionHistory represents monthly active vs cancelled subscriptions
type SubscriptionHistory struct {
    Month     string `json:"month"`
    Active    int    `json:"active"`
    Cancelled int    `json:"cancelled"`
}
