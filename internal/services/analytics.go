package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnalyticsService struct {
	db *gorm.DB
}

func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

type OverviewStats struct {
	TotalBids    int
	Won          int
	Lost         int
	WinRate      float64
	AvgPrice     float64
	AvgHours     float64
	TotalRevenue float64
	AvgResponse  float64 // hours between created and submitted
}

type PriceBracket struct {
	Label   string
	Won     int
	Lost    int
	Total   int
	WinRate float64
}

type MonthlyTrend struct {
	Month    string // "Jan 2026"
	Bids     int
	Won      int
	WinRate  float64
	Revenue  float64
	AvgPrice float64
}

type TemplateStats struct {
	Name    string
	Uses    int
	Wins    int
	WinRate float64
}

func (s *AnalyticsService) Overview(userID uuid.UUID, since time.Time) OverviewStats {
	var stats OverviewStats

	base := s.db.Table("bids").Where("user_id = ? AND created_at >= ?", userID, since)

	var totalCount, wonCount, lostCount int64
	base.Count(&totalCount)
	stats.TotalBids = int(totalCount)

	s.db.Table("bids").Where("user_id = ? AND created_at >= ? AND status = 'won'", userID, since).
		Count(&wonCount)
	stats.Won = int(wonCount)
	s.db.Table("bids").Where("user_id = ? AND created_at >= ? AND status = 'lost'", userID, since).
		Count(&lostCount)
	stats.Lost = int(lostCount)

	decided := stats.Won + stats.Lost
	if decided > 0 {
		stats.WinRate = float64(stats.Won) / float64(decided) * 100
	}

	var avgResult struct {
		AvgPrice float64
		AvgHours float64
	}
	base.Select("COALESCE(AVG(total_price), 0) as avg_price, COALESCE(AVG(estimated_hours), 0) as avg_hours").
		Scan(&avgResult)
	stats.AvgPrice = avgResult.AvgPrice
	stats.AvgHours = avgResult.AvgHours

	var rev struct{ Total float64 }
	s.db.Table("bids").Where("user_id = ? AND created_at >= ? AND status = 'won'", userID, since).
		Select("COALESCE(SUM(total_price), 0) as total").Scan(&rev)
	stats.TotalRevenue = rev.Total

	// Average response time (hours)
	var avgResp struct{ Hours float64 }
	s.db.Table("bids").
		Where("user_id = ? AND created_at >= ? AND submitted_at IS NOT NULL", userID, since).
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (submitted_at - created_at)) / 3600), 0) as hours").
		Scan(&avgResp)
	stats.AvgResponse = avgResp.Hours

	return stats
}

func (s *AnalyticsService) ByPriceBracket(userID uuid.UUID, since time.Time) []PriceBracket {
	brackets := []struct {
		label string
		min   float64
		max   float64
	}{
		{"< $500", 0, 500},
		{"$500 – $2k", 500, 2000},
		{"$2k – $5k", 2000, 5000},
		{"$5k – $10k", 5000, 10000},
		{"$10k+", 10000, 1e9},
	}

	results := make([]PriceBracket, len(brackets))
	for i, b := range brackets {
		var won, lost int64
		s.db.Table("bids").
			Where("user_id = ? AND created_at >= ? AND total_price >= ? AND total_price < ? AND status = 'won'",
				userID, since, b.min, b.max).Count(&won)
		s.db.Table("bids").
			Where("user_id = ? AND created_at >= ? AND total_price >= ? AND total_price < ? AND status = 'lost'",
				userID, since, b.min, b.max).Count(&lost)

		total := int(won + lost)
		winRate := 0.0
		if total > 0 {
			winRate = float64(won) / float64(total) * 100
		}
		results[i] = PriceBracket{
			Label:   b.label,
			Won:     int(won),
			Lost:    int(lost),
			Total:   total,
			WinRate: winRate,
		}
	}
	return results
}

func (s *AnalyticsService) Trends(userID uuid.UUID, since time.Time) []MonthlyTrend {
	type row struct {
		Month    time.Time
		Bids     int
		Won      int
		Revenue  float64
		AvgPrice float64
	}

	var rows []row
	s.db.Table("bids").
		Where("user_id = ? AND created_at >= ?", userID, since).
		Select(`
			date_trunc('month', created_at) as month,
			COUNT(*) as bids,
			COUNT(*) FILTER (WHERE status = 'won') as won,
			COALESCE(SUM(total_price) FILTER (WHERE status = 'won'), 0) as revenue,
			COALESCE(AVG(total_price), 0) as avg_price
		`).
		Group("month").
		Order("month").
		Scan(&rows)

	trends := make([]MonthlyTrend, len(rows))
	for i, r := range rows {
		winRate := 0.0
		if r.Bids > 0 {
			winRate = float64(r.Won) / float64(r.Bids) * 100
		}
		trends[i] = MonthlyTrend{
			Month:    r.Month.Format("Jan 2006"),
			Bids:     r.Bids,
			Won:      r.Won,
			WinRate:  winRate,
			Revenue:  r.Revenue,
			AvgPrice: r.AvgPrice,
		}
	}
	return trends
}

func (s *AnalyticsService) TemplateEffectiveness(userID uuid.UUID, since time.Time) []TemplateStats {
	type row struct {
		Name string
		Uses int
		Wins int
	}

	var rows []row
	s.db.Table("templates").
		Where("templates.user_id = ? AND templates.created_at >= ?", userID, since).
		Select("templates.name, templates.use_count as uses, templates.win_count as wins").
		Order("templates.win_count desc").
		Scan(&rows)

	// Also get stats for bids with no template
	var noTmpl struct {
		Uses int
		Wins int
	}
	s.db.Table("bids").
		Where("user_id = ? AND created_at >= ? AND template_id IS NULL AND status IN ('won','lost')", userID, since).
		Select("COUNT(*) as uses, COUNT(*) FILTER (WHERE status = 'won') as wins").
		Scan(&noTmpl)

	results := make([]TemplateStats, 0, len(rows)+1)
	for _, r := range rows {
		winRate := 0.0
		if r.Uses > 0 {
			winRate = float64(r.Wins) / float64(r.Uses) * 100
		}
		results = append(results, TemplateStats{
			Name:    r.Name,
			Uses:    r.Uses,
			Wins:    r.Wins,
			WinRate: winRate,
		})
	}

	if noTmpl.Uses > 0 {
		winRate := float64(noTmpl.Wins) / float64(noTmpl.Uses) * 100
		results = append(results, TemplateStats{
			Name:    "No template",
			Uses:    noTmpl.Uses,
			Wins:    noTmpl.Wins,
			WinRate: winRate,
		})
	}

	return results
}
