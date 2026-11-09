CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS cards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_number VARCHAR(16) UNIQUE NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'blocked')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stations (
    id VARCHAR(32) PRIMARY KEY, -- e.g. 'MRT-DKA', 'TJ-HBI'
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(150) UNIQUE NOT NULL,
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('mrt', 'lrt', 'transjakarta', 'mikrotrans')),
    zone_index INT NOT NULL DEFAULT 1,
    city VARCHAR(100) NOT NULL DEFAULT 'dki jakarta',
    district VARCHAR(100) NOT NULL DEFAULT '',
    subdistrict VARCHAR(100) NOT NULL DEFAULT '',
    latitude NUMERIC(10, 7),
    longitude NUMERIC(10, 7),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS routes (
    id VARCHAR(32) PRIMARY KEY, -- e.g. 'MRT-M1', 'TJ-1', 'LRT-BK'
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(200) UNIQUE NOT NULL,
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('mrt', 'lrt', 'transjakarta', 'mikrotrans')),
    color_hex VARCHAR(7) DEFAULT '#000000',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS route_stations (
    route_id VARCHAR(32) NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    station_id VARCHAR(32) NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    stop_sequence INT NOT NULL CHECK (stop_sequence > 0),
    distance_from_origin_km NUMERIC(5, 2) DEFAULT 0.0 CHECK (distance_from_origin_km >= 0),
    direction VARCHAR(20) NOT NULL DEFAULT 'bidirectional' CHECK (direction IN ('bidirectional', 'outbound', 'inbound')),
    PRIMARY KEY (route_id, station_id)
);

CREATE TABLE IF NOT EXISTS journeys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE RESTRICT,
    entry_station_id VARCHAR(32) NOT NULL REFERENCES stations(id),
    exit_station_id VARCHAR(32) REFERENCES stations(id),
    route_id VARCHAR(32) REFERENCES routes(id) ON DELETE SET NULL,
    entry_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    exit_time TIMESTAMP WITH TIME ZONE,
    fare_amount BIGINT DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed', 'penalty_settled', 'expired')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ledger_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE RESTRICT,
    journey_id UUID REFERENCES journeys(id) ON DELETE SET NULL,
    transaction_type VARCHAR(30) NOT NULL CHECK (transaction_type IN ('top_up', 'fare_deduction', 'penalty_fee', 'refund')),
    amount BIGINT NOT NULL, -- Positive for credit, negative for debit
    balance_after BIGINT NOT NULL,
    idempotency_key VARCHAR(100) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cards_card_number ON cards(card_number);
CREATE INDEX IF NOT EXISTS idx_stations_slug ON stations(slug);
CREATE INDEX IF NOT EXISTS idx_stations_city ON stations(city);
CREATE INDEX IF NOT EXISTS idx_stations_district ON stations(district);
CREATE INDEX IF NOT EXISTS idx_stations_mode_city ON stations(mode, city);
CREATE INDEX IF NOT EXISTS idx_routes_slug ON routes(slug);
CREATE INDEX IF NOT EXISTS idx_route_stations_station ON route_stations(station_id);
CREATE INDEX IF NOT EXISTS idx_route_stations_order ON route_stations(route_id, stop_sequence);
CREATE INDEX IF NOT EXISTS idx_journeys_active ON journeys(card_id) WHERE status = 'in_progress';
CREATE INDEX IF NOT EXISTS idx_journeys_route_id ON journeys(route_id);
CREATE INDEX IF NOT EXISTS idx_ledger_card_id ON ledger_transactions(card_id);

INSERT INTO stations (id, name, slug, mode, zone_index, city, district, subdistrict, latitude, longitude) VALUES
('MRT-BHI', 'Bundaran HI', 'bundaran-hi', 'mrt', 1, 'jakarta pusat', 'menteng', 'gondangdia', -6.1925, 106.8236),
('MRT-DKA', 'Dukuh Atas BNI', 'dukuh-atas-bni', 'mrt', 1, 'jakarta pusat', 'tanah abang', 'kebon melati', -6.2008, 106.8228),
('MRT-BLB', 'Lebak Bulus Grab', 'lebak-bulus-grab', 'mrt', 3, 'jakarta selatan', 'cilandak', 'lebak bulus', -6.2892, 106.7744),
('TJ-KOTA', 'Kota', 'kota', 'transjakarta', 1, 'jakarta barat', 'taman sari', 'pinangsia', -6.1376, 106.8146),
('TJ-MNAS', 'Monas', 'monas', 'transjakarta', 1, 'jakarta pusat', 'gambir', 'gambir', -6.1754, 106.8272),
('TJ-HRMN', 'Harmoni', 'harmoni', 'transjakarta', 1, 'jakarta pusat', 'gambir', 'kebon kelapa', -6.1661, 106.8202),
('LRT-DKA', 'Dukuh Atas LRT', 'dukuh-atas-lrt', 'lrt', 1, 'jakarta selatan', 'setiabudi', 'setiabudi', -6.2091, 106.8285),
('LRT-JAT', 'Jatimulya', 'jatimulya', 'lrt', 4, 'kota bekasi', 'bekasi timur', 'jatimulya', -6.2625, 107.0175)
ON CONFLICT (id) DO NOTHING;

INSERT INTO routes (id, code, name, slug, mode, color_hex, is_active) VALUES
('MRT-M1', 'M1', 'MRT North-South Line (Lebak Bulus - Bundaran HI)', 'mrt-north-south-line-lebak-bulus-bundaran-hi', 'mrt', '#1e439b', TRUE),
('TJ-1', '1', 'Koridor 1 (Blok M - Kota)', 'koridor-1-blok-m-kota', 'transjakarta', '#e30613', TRUE),
('LRT-BK', 'BK', 'LRT Bekasi Line (Dukuh Atas - Jatimulya)', 'lrt-bekasi-line-dukuh-atas-jatimulya', 'lrt', '#d4145a', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO route_stations (route_id, station_id, stop_sequence, distance_from_origin_km) VALUES
('MRT-M1', 'MRT-BLB', 1, 0.0),
('MRT-M1', 'MRT-DKA', 2, 11.2),
('MRT-M1', 'MRT-BHI', 3, 15.7),
('TJ-1', 'TJ-MNAS', 1, 10.5),
('TJ-1', 'TJ-HRMN', 2, 12.0),
('TJ-1', 'TJ-KOTA', 3, 15.6),
('LRT-BK', 'LRT-DKA', 1, 0.0),
('LRT-BK', 'LRT-JAT', 2, 27.3)
ON CONFLICT (route_id, station_id) DO NOTHING;
