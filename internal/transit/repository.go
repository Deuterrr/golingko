// Package transit orchestrates commuter journeys, turnstile gate validation, and transit card lifecycles.
// It assesses multi-modal JakLingko fare calculations and guarantees accurate ledger settlements across transit networks.
package transit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golingko/pkg/database"
)

// Common query execution abstraction.
type DBExecutor = database.DBExecutor

type CardRecord struct {
	ID         string
	CardNumber string
	Balance    int64
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type StationRecord struct {
	ID          string
	Name        string
	Slug        string
	Mode        string
	ZoneIndex   int
	City        string
	District    string
	Subdistrict string
	Latitude    *float64
	Longitude   *float64
	IsActive    bool
}

type StationFilterRecord struct {
	Mode     *string
	City     *string
	District *string
	RouteID  *string
}

type RouteRecord struct {
	ID        string
	Code      string
	Name      string
	Slug      string
	Mode      string
	ColorHex  string
	IsActive  bool
	CreatedAt time.Time
}

type RouteStopRecord struct {
	RouteID              string
	StationID            string
	StationName          string
	StationSlug          string
	Mode                 string
	City                 string
	District             string
	Subdistrict          string
	StopSequence         int
	DistanceFromOriginKm float64
	Direction            string
}

type JourneyRecord struct {
	ID             string
	CardID         string
	EntryStationID string
	ExitStationID  *string
	RouteID        *string
	EntryTime      time.Time
	ExitTime       *time.Time
	FareAmount     int64
	Status         string
	CreatedAt      time.Time
}

type LedgerRecord struct {
	ID              string
	CardID          string
	JourneyID       *string
	TransactionType string
	Amount          int64
	BalanceAfter    int64
	IdempotencyKey  *string
	CreatedAt       time.Time
}

type CompletedLegDTO struct {
	Mode       string
	FareAmount int64
	ExitTime   time.Time
}

type ExpiredTripDTO struct {
	JourneyID string
	CardID    string
	Balance   int64
}

// Persistence contract.
type TransitRepository interface {
	WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error

	// Cards
	CreateCard(ctx context.Context, exec DBExecutor, cardNumber string, initialBalance int64, status string) (*CardRecord, error)
	GetCardByNumber(ctx context.Context, exec DBExecutor, number string) (*CardRecord, error)
	GetCardForUpdate(ctx context.Context, tx *sql.Tx, number string) (*CardRecord, error)
	UpdateCardBalance(ctx context.Context, tx *sql.Tx, cardID string, balance int64) error
	UpdateCardStatus(ctx context.Context, tx *sql.Tx, cardID string, status string) error

	// Stations
	ListStations(ctx context.Context, exec DBExecutor, filter StationFilterRecord) ([]StationRecord, error)
	GetStationByID(ctx context.Context, exec DBExecutor, id string) (*StationRecord, error)
	GetStationRoutes(ctx context.Context, exec DBExecutor, stationID string) ([]RouteRecord, error)

	// Routes
	ListRoutes(ctx context.Context, exec DBExecutor, mode *string, city *string) ([]RouteRecord, error)
	GetRouteByID(ctx context.Context, exec DBExecutor, id string) (*RouteRecord, []RouteStopRecord, error)
	GetConnectingRoutes(ctx context.Context, exec DBExecutor, originStationID, destStationID string) ([]RouteRecord, error)
	IsStationOnRoute(ctx context.Context, exec DBExecutor, routeID, stationID string) (bool, error)

	// Journeys
	GetActiveJourneyForUpdate(ctx context.Context, tx *sql.Tx, cardID string) (*JourneyRecord, error)
	CreateJourney(ctx context.Context, tx *sql.Tx, cardID, stationID string, routeID *string) (*JourneyRecord, error)
	CompleteJourney(ctx context.Context, tx *sql.Tx, journeyID, exitStationID string, exitTime time.Time, fare int64) error
	ExpireJourney(ctx context.Context, tx *sql.Tx, journeyID string, penalty int64) error
	ListRecentCompletedJourneys(ctx context.Context, exec DBExecutor, cardID string, since time.Time) ([]CompletedLegDTO, error)
	ListExpiredInProgressJourneys(ctx context.Context, exec DBExecutor, threshold time.Time, limit int) ([]ExpiredTripDTO, error)

	// Ledger
	RecordLedgerTransaction(ctx context.Context, tx *sql.Tx, entry LedgerRecord) (*LedgerRecord, error)
	ListLedgerTransactions(ctx context.Context, exec DBExecutor, cardID string, limit int) ([]LedgerRecord, error)
}

type sqlRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) TransitRepository {
	return &sqlRepository{db: db}
}

func (r *sqlRepository) getExecutor(exec DBExecutor) (DBExecutor, error) {
	if exec != nil {
		return exec, nil
	}
	if r.db == nil {
		return nil, errors.New("repository database connection is nil")
	}
	return r.db, nil
}

func (r *sqlRepository) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return database.WithTransaction(ctx, r.db, fn)
}

func (r *sqlRepository) CreateCard(ctx context.Context, exec DBExecutor, cardNumber string, initialBalance int64, status string) (*CardRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO cards (card_number, balance, status)
		VALUES ($1, $2, $3)
		RETURNING id, card_number, balance, status, created_at, updated_at
	`

	var record CardRecord
	err = executor.QueryRowContext(ctx, query, cardNumber, initialBalance, status).Scan(
		&record.ID,
		&record.CardNumber,
		&record.Balance,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert card: %w", err)
	}

	return &record, nil
}

func (r *sqlRepository) GetCardByNumber(ctx context.Context, exec DBExecutor, number string) (*CardRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, card_number, balance, status, created_at, updated_at
		FROM cards
		WHERE card_number = $1
	`

	var record CardRecord
	err = executor.QueryRowContext(ctx, query, number).Scan(
		&record.ID,
		&record.CardNumber,
		&record.Balance,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to query card by number: %w", err)
	}

	return &record, nil
}

func (r *sqlRepository) GetCardForUpdate(ctx context.Context, tx *sql.Tx, number string) (*CardRecord, error) {
	query := `
		SELECT id, card_number, balance, status, created_at, updated_at
		FROM cards
		WHERE card_number = $1
		FOR UPDATE
	`

	var record CardRecord
	err := tx.QueryRowContext(ctx, query, number).Scan(
		&record.ID,
		&record.CardNumber,
		&record.Balance,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to lock card: %w", err)
	}

	return &record, nil
}

func (r *sqlRepository) UpdateCardBalance(ctx context.Context, tx *sql.Tx, cardID string, balance int64) error {
	if balance < 0 {
		return ErrNegativeBalance
	}

	query := `
		UPDATE cards
		SET balance = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := tx.ExecContext(ctx, query, balance, cardID)
	if err != nil {
		return fmt.Errorf("failed to update card balance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCardNotFound
	}

	return nil
}

func (r *sqlRepository) UpdateCardStatus(ctx context.Context, tx *sql.Tx, cardID string, status string) error {
	query := `
		UPDATE cards
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := tx.ExecContext(ctx, query, status, cardID)
	return err
}

func (r *sqlRepository) ListStations(ctx context.Context, exec DBExecutor, filter StationFilterRecord) ([]StationRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT s.id, s.name, s.slug, s.mode, s.zone_index, s.city, s.district, s.subdistrict, s.latitude, s.longitude, s.is_active
		FROM stations s
	`
	var conditions []string
	var args []any
	argIndex := 1

	if filter.RouteID != nil && *filter.RouteID != "" {
		query += fmt.Sprintf(" INNER JOIN route_stations rs ON rs.station_id = s.id AND rs.route_id = $%d", argIndex)
		args = append(args, strings.TrimSpace(*filter.RouteID))
		argIndex++
	}

	conditions = append(conditions, "s.is_active = TRUE")

	if filter.Mode != nil && *filter.Mode != "" {
		conditions = append(conditions, fmt.Sprintf("s.mode = $%d", argIndex))
		args = append(args, strings.ToLower(strings.TrimSpace(*filter.Mode)))
		argIndex++
	}
	if filter.City != nil && *filter.City != "" {
		conditions = append(conditions, fmt.Sprintf("s.city = $%d", argIndex))
		args = append(args, strings.ToLower(strings.TrimSpace(*filter.City)))
		argIndex++
	}
	if filter.District != nil && *filter.District != "" {
		conditions = append(conditions, fmt.Sprintf("s.district = $%d", argIndex))
		args = append(args, strings.ToLower(strings.TrimSpace(*filter.District)))
		argIndex++
	}

	query += " WHERE " + strings.Join(conditions, " AND ")
	if filter.RouteID != nil && *filter.RouteID != "" {
		query += " ORDER BY rs.stop_sequence ASC"
	} else {
		query += " ORDER BY s.mode, s.name ASC"
	}

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list stations: %w", err)
	}
	defer rows.Close()

	var stations []StationRecord
	for rows.Next() {
		var st StationRecord
		if err := rows.Scan(
			&st.ID,
			&st.Name,
			&st.Slug,
			&st.Mode,
			&st.ZoneIndex,
			&st.City,
			&st.District,
			&st.Subdistrict,
			&st.Latitude,
			&st.Longitude,
			&st.IsActive,
		); err != nil {
			return nil, err
		}
		stations = append(stations, st)
	}

	return stations, nil
}

func (r *sqlRepository) GetStationByID(ctx context.Context, exec DBExecutor, id string) (*StationRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, name, slug, mode, zone_index, city, district, subdistrict, latitude, longitude, is_active
		FROM stations
		WHERE id = $1 OR slug = $1
	`

	var st StationRecord
	err = executor.QueryRowContext(ctx, query, id).Scan(
		&st.ID,
		&st.Name,
		&st.Slug,
		&st.Mode,
		&st.ZoneIndex,
		&st.City,
		&st.District,
		&st.Subdistrict,
		&st.Latitude,
		&st.Longitude,
		&st.IsActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStationNotFound
		}
		return nil, fmt.Errorf("failed to get station: %w", err)
	}

	return &st, nil
}

func (r *sqlRepository) GetStationRoutes(ctx context.Context, exec DBExecutor, stationID string) ([]RouteRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT r.id, r.code, r.name, r.slug, r.mode, r.color_hex, r.is_active, r.created_at
		FROM routes r
		INNER JOIN route_stations rs ON rs.route_id = r.id
		WHERE rs.station_id = $1 AND r.is_active = TRUE
		ORDER BY r.code ASC
	`
	rows, err := executor.QueryContext(ctx, query, stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get station routes: %w", err)
	}
	defer rows.Close()

	var routes []RouteRecord
	for rows.Next() {
		var rec RouteRecord
		if err := rows.Scan(&rec.ID, &rec.Code, &rec.Name, &rec.Slug, &rec.Mode, &rec.ColorHex, &rec.IsActive, &rec.CreatedAt); err != nil {
			return nil, err
		}
		routes = append(routes, rec)
	}
	return routes, nil
}

func (r *sqlRepository) ListRoutes(ctx context.Context, exec DBExecutor, mode *string, city *string) ([]RouteRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT DISTINCT r.id, r.code, r.name, r.slug, r.mode, r.color_hex, r.is_active, r.created_at
		FROM routes r
	`
	var conditions []string
	var args []any
	argIndex := 1

	if city != nil && *city != "" {
		query += fmt.Sprintf(" INNER JOIN route_stations rs ON rs.route_id = r.id INNER JOIN stations s ON s.id = rs.station_id AND s.city = $%d", argIndex)
		args = append(args, strings.ToLower(strings.TrimSpace(*city)))
		argIndex++
	}

	conditions = append(conditions, "r.is_active = TRUE")

	if mode != nil && *mode != "" {
		conditions = append(conditions, fmt.Sprintf("r.mode = $%d", argIndex))
		args = append(args, strings.ToLower(strings.TrimSpace(*mode)))
		argIndex++
	}

	query += " WHERE " + strings.Join(conditions, " AND ")
	query += " ORDER BY r.code ASC"

	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}
	defer rows.Close()

	var routes []RouteRecord
	for rows.Next() {
		var rec RouteRecord
		if err := rows.Scan(&rec.ID, &rec.Code, &rec.Name, &rec.Slug, &rec.Mode, &rec.ColorHex, &rec.IsActive, &rec.CreatedAt); err != nil {
			return nil, err
		}
		routes = append(routes, rec)
	}
	return routes, nil
}

func (r *sqlRepository) GetRouteByID(ctx context.Context, exec DBExecutor, id string) (*RouteRecord, []RouteStopRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, nil, err
	}

	routeQuery := `
		SELECT id, code, name, slug, mode, color_hex, is_active, created_at
		FROM routes
		WHERE id = $1 OR slug = $1
	`
	var rec RouteRecord
	err = executor.QueryRowContext(ctx, routeQuery, id).Scan(
		&rec.ID, &rec.Code, &rec.Name, &rec.Slug, &rec.Mode, &rec.ColorHex, &rec.IsActive, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrRouteNotFound
		}
		return nil, nil, fmt.Errorf("failed to get route %q: %w", id, err)
	}

	stopsQuery := `
		SELECT rs.route_id, rs.station_id, s.name, s.slug, s.mode, s.city, s.district, s.subdistrict, rs.stop_sequence, rs.distance_from_origin_km, rs.direction
		FROM route_stations rs
		INNER JOIN stations s ON s.id = rs.station_id
		WHERE rs.route_id = $1
		ORDER BY rs.stop_sequence ASC
	`
	rows, err := executor.QueryContext(ctx, stopsQuery, rec.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get route stops: %w", err)
	}
	defer rows.Close()

	var stops []RouteStopRecord
	for rows.Next() {
		var stop RouteStopRecord
		if err := rows.Scan(
			&stop.RouteID,
			&stop.StationID,
			&stop.StationName,
			&stop.StationSlug,
			&stop.Mode,
			&stop.City,
			&stop.District,
			&stop.Subdistrict,
			&stop.StopSequence,
			&stop.DistanceFromOriginKm,
			&stop.Direction,
		); err != nil {
			return nil, nil, err
		}
		stops = append(stops, stop)
	}

	return &rec, stops, nil
}

func (r *sqlRepository) GetConnectingRoutes(ctx context.Context, exec DBExecutor, originStationID, destStationID string) ([]RouteRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT r.id, r.code, r.name, r.slug, r.mode, r.color_hex, r.is_active, r.created_at
		FROM routes r
		INNER JOIN route_stations rs1 ON rs1.route_id = r.id AND rs1.station_id = $1
		INNER JOIN route_stations rs2 ON rs2.route_id = r.id AND rs2.station_id = $2
		WHERE r.is_active = TRUE
		ORDER BY r.code ASC
	`
	rows, err := executor.QueryContext(ctx, query, originStationID, destStationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connecting routes: %w", err)
	}
	defer rows.Close()

	var routes []RouteRecord
	for rows.Next() {
		var rec RouteRecord
		if err := rows.Scan(&rec.ID, &rec.Code, &rec.Name, &rec.Slug, &rec.Mode, &rec.ColorHex, &rec.IsActive, &rec.CreatedAt); err != nil {
			return nil, err
		}
		routes = append(routes, rec)
	}
	return routes, nil
}

func (r *sqlRepository) IsStationOnRoute(ctx context.Context, exec DBExecutor, routeID, stationID string) (bool, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return false, err
	}

	query := `
		SELECT 1
		FROM route_stations rs
		INNER JOIN routes r ON r.id = rs.route_id
		WHERE rs.route_id = $1 AND rs.station_id = $2 AND r.is_active = TRUE
		LIMIT 1
	`
	var dummy int
	err = executor.QueryRowContext(ctx, query, routeID, stationID).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check station on route: %w", err)
	}
	return true, nil
}

func (r *sqlRepository) GetActiveJourneyForUpdate(ctx context.Context, tx *sql.Tx, cardID string) (*JourneyRecord, error) {
	query := `
		SELECT id, card_id, entry_station_id, exit_station_id, route_id, entry_time, exit_time, fare_amount, status, created_at
		FROM journeys
		WHERE card_id = $1 AND status = 'in_progress'
		ORDER BY entry_time DESC
		LIMIT 1
		FOR UPDATE
	`

	var j JourneyRecord
	err := tx.QueryRowContext(ctx, query, cardID).Scan(
		&j.ID,
		&j.CardID,
		&j.EntryStationID,
		&j.ExitStationID,
		&j.RouteID,
		&j.EntryTime,
		&j.ExitTime,
		&j.FareAmount,
		&j.Status,
		&j.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoActiveJourney
		}
		return nil, fmt.Errorf("failed to query active journey: %w", err)
	}

	return &j, nil
}

func (r *sqlRepository) CreateJourney(ctx context.Context, tx *sql.Tx, cardID, stationID string, routeID *string) (*JourneyRecord, error) {
	query := `
		INSERT INTO journeys (card_id, entry_station_id, route_id, status)
		VALUES ($1, $2, $3, 'in_progress')
		RETURNING id, card_id, entry_station_id, exit_station_id, route_id, entry_time, exit_time, fare_amount, status, created_at
	`

	var j JourneyRecord
	err := tx.QueryRowContext(ctx, query, cardID, stationID, routeID).Scan(
		&j.ID,
		&j.CardID,
		&j.EntryStationID,
		&j.ExitStationID,
		&j.RouteID,
		&j.EntryTime,
		&j.ExitTime,
		&j.FareAmount,
		&j.Status,
		&j.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create journey: %w", err)
	}

	return &j, nil
}

func (r *sqlRepository) CompleteJourney(ctx context.Context, tx *sql.Tx, journeyID, exitStationID string, exitTime time.Time, fare int64) error {
	query := `
		UPDATE journeys
		SET exit_station_id = $1, exit_time = $2, fare_amount = $3, status = 'completed'
		WHERE id = $4
	`
	_, err := tx.ExecContext(ctx, query, exitStationID, exitTime, fare, journeyID)
	return err
}

func (r *sqlRepository) ExpireJourney(ctx context.Context, tx *sql.Tx, journeyID string, penalty int64) error {
	query := `
		UPDATE journeys
		SET status = 'expired', fare_amount = $1, exit_time = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = 'in_progress'
	`
	_, err := tx.ExecContext(ctx, query, penalty, journeyID)
	return err
}

func (r *sqlRepository) ListRecentCompletedJourneys(ctx context.Context, exec DBExecutor, cardID string, since time.Time) ([]CompletedLegDTO, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT s.mode, j.fare_amount, j.exit_time
		FROM journeys j
		JOIN stations s ON j.entry_station_id = s.id
		WHERE j.card_id = $1
		  AND j.status = 'completed'
		  AND j.exit_time >= $2
		ORDER BY j.exit_time ASC
	`

	rows, err := executor.QueryContext(ctx, query, cardID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query transfer window trips: %w", err)
	}
	defer rows.Close()

	var legs []CompletedLegDTO
	for rows.Next() {
		var leg CompletedLegDTO
		if err := rows.Scan(&leg.Mode, &leg.FareAmount, &leg.ExitTime); err == nil {
			legs = append(legs, leg)
		}
	}

	return legs, nil
}

func (r *sqlRepository) ListExpiredInProgressJourneys(ctx context.Context, exec DBExecutor, threshold time.Time, limit int) ([]ExpiredTripDTO, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT j.id, j.card_id, c.balance
		FROM journeys j
		JOIN cards c ON j.card_id = c.id
		WHERE j.status = 'in_progress' AND j.entry_time < $1
		LIMIT $2
	`

	rows, err := executor.QueryContext(ctx, query, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired journeys: %w", err)
	}
	defer rows.Close()

	var trips []ExpiredTripDTO
	for rows.Next() {
		var t ExpiredTripDTO
		if err := rows.Scan(&t.JourneyID, &t.CardID, &t.Balance); err == nil {
			trips = append(trips, t)
		}
	}

	return trips, nil
}

func (r *sqlRepository) RecordLedgerTransaction(ctx context.Context, tx *sql.Tx, entry LedgerRecord) (*LedgerRecord, error) {
	query := `
		INSERT INTO ledger_transactions (card_id, journey_id, transaction_type, amount, balance_after, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, card_id, journey_id, transaction_type, amount, balance_after, idempotency_key, created_at
	`

	var recorded LedgerRecord
	err := tx.QueryRowContext(ctx, query,
		entry.CardID,
		entry.JourneyID,
		entry.TransactionType,
		entry.Amount,
		entry.BalanceAfter,
		entry.IdempotencyKey,
	).Scan(
		&recorded.ID,
		&recorded.CardID,
		&recorded.JourneyID,
		&recorded.TransactionType,
		&recorded.Amount,
		&recorded.BalanceAfter,
		&recorded.IdempotencyKey,
		&recorded.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record ledger transaction: %w", err)
	}

	return &recorded, nil
}

func (r *sqlRepository) ListLedgerTransactions(ctx context.Context, exec DBExecutor, cardID string, limit int) ([]LedgerRecord, error) {
	executor, err := r.getExecutor(exec)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := `
		SELECT id, card_id, journey_id, transaction_type, amount, balance_after, idempotency_key, created_at
		FROM ledger_transactions
		WHERE card_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := executor.QueryContext(ctx, query, cardID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []LedgerRecord
	for rows.Next() {
		var txItem LedgerRecord
		if err := rows.Scan(
			&txItem.ID,
			&txItem.CardID,
			&txItem.JourneyID,
			&txItem.TransactionType,
			&txItem.Amount,
			&txItem.BalanceAfter,
			&txItem.IdempotencyKey,
			&txItem.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, txItem)
	}

	return transactions, nil
}
