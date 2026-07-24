package hyperpaths

// Link is an edge in the transit network graph.
type Link struct {
	// Source node of the link
	FromNode string
	// Target node of the link
	ToNode string
	// Corresponding route
	RouteID string
	// Travel time along the link (in minutes or any consistent unit)
	TravelCost float64
	// Service headway. Boarding links have headway > 0 (frequency = 1/headway).
	// On-board (riding) links have headway = 0 (no waiting).
	Headway float64
}
