-- Transit Stations
INSERT INTO stations (id, name, slug, mode, zone_index, city, district, subdistrict, latitude, longitude, is_active) VALUES
-- MRT Jakarta (North-South Heavy Rail)
('MRT-BHI', 'Bundaran HI', 'bundaran-hi', 'mrt', 1, 'jakarta pusat', 'menteng', 'gondangdia', -6.1925, 106.8236, TRUE),
('MRT-DKA', 'Dukuh Atas BNI', 'dukuh-atas-bni', 'mrt', 1, 'jakarta pusat', 'tanah abang', 'kebon melati', -6.2008, 106.8228, TRUE),
('MRT-BLM', 'Blok M BCA', 'blok-m-bca', 'mrt', 2, 'jakarta selatan', 'kebayoran baru', 'melawai', -6.2444, 106.7981, TRUE),
('MRT-BLB', 'Lebak Bulus Grab', 'lebak-bulus-grab', 'mrt', 3, 'jakarta selatan', 'cilandak', 'lebak bulus', -6.2892, 106.7744, TRUE),

-- TransJakarta (25 BRT Stations across Corridors 1, 2, and 6)
('TJ-BLKM', 'Blok M', 'blok-m', 'transjakarta', 2, 'jakarta selatan', 'kebayoran baru', 'melawai', -6.2442, 106.7983, TRUE),
('TJ-CSW', 'CSW / ASEAN', 'csw-asean', 'transjakarta', 2, 'jakarta selatan', 'kebayoran baru', 'selong', -6.2396, 106.7989, TRUE),
('TJ-BNSN', 'Bundaran Senayan', 'bundaran-senayan', 'transjakarta', 1, 'jakarta selatan', 'kebayoran baru', 'senayan', -6.2272, 106.8025, TRUE),
('TJ-GBK', 'Gelora Bung Karno', 'gelora-bung-karno', 'transjakarta', 1, 'jakarta pusat', 'tanah abang', 'gelora', -6.2185, 106.8058, TRUE),
('TJ-PLDA', 'Polda Metro Jaya', 'polda-metro-jaya', 'transjakarta', 1, 'jakarta selatan', 'kebayoran baru', 'senayan', -6.2235, 106.8105, TRUE),
('TJ-BDHL', 'Bendungan Hilir', 'bendungan-hilir', 'transjakarta', 1, 'jakarta pusat', 'tanah abang', 'karet tengsin', -6.2155, 106.8188, TRUE),
('TJ-KART', 'Karet Sudirman', 'karet-sudirman', 'transjakarta', 1, 'jakarta pusat', 'tanah abang', 'karet tengsin', -6.2088, 106.8205, TRUE),
('TJ-TOSR', 'Tosari ICBC', 'tosari-icbc', 'transjakarta', 1, 'jakarta pusat', 'menteng', 'gondangdia', -6.2005, 106.8230, TRUE),
('TJ-BNHI', 'Bundaran HI Astra', 'bundaran-hi-astra', 'transjakarta', 1, 'jakarta pusat', 'menteng', 'gondangdia', -6.1945, 106.8232, TRUE),
('TJ-SRNH', 'M.H. Thamrin / Sarinah', 'mh-thamrin-sarinah', 'transjakarta', 1, 'jakarta pusat', 'menteng', 'gondangdia', -6.1878, 106.8240, TRUE),
('TJ-BKID', 'Kebon Sirih / Bank Indonesia', 'kebon-sirih-bank-indonesia', 'transjakarta', 1, 'jakarta pusat', 'gambir', 'gambir', -6.1812, 106.8248, TRUE),
('TJ-MNAS', 'Monas', 'monas', 'transjakarta', 1, 'jakarta pusat', 'gambir', 'gambir', -6.1754, 106.8272, TRUE),
('TJ-HRMN', 'Harmoni', 'harmoni', 'transjakarta', 1, 'jakarta pusat', 'gambir', 'kebon kelapa', -6.1661, 106.8202, TRUE),
('TJ-MGBS', 'Mangga Besar', 'mangga-besar', 'transjakarta', 1, 'jakarta barat', 'taman sari', 'tangki', -6.1495, 106.8185, TRUE),
('TJ-GLDK', 'Glodok', 'glodok', 'transjakarta', 1, 'jakarta barat', 'taman sari', 'glodok', -6.1432, 106.8160, TRUE),
('TJ-KOTA', 'Kota', 'kota', 'transjakarta', 1, 'jakarta barat', 'taman sari', 'pinangsia', -6.1376, 106.8146, TRUE),
('TJ-RAGN', 'Ragunan', 'ragunan', 'transjakarta', 2, 'jakarta selatan', 'pasar minggu', 'ragunan', -6.3117, 106.8206, TRUE),
('TJ-JATI', 'Jati Padang', 'jati-padang', 'transjakarta', 2, 'jakarta selatan', 'pasar minggu', 'jati padang', -6.2925, 106.8272, TRUE),
('TJ-PJT', 'Pejaten', 'pejaten', 'transjakarta', 2, 'jakarta selatan', 'pasar minggu', 'pejaten barat', -6.2755, 106.8285, TRUE),
('TJ-BUNC', 'Buncit Indah', 'buncit-indah', 'transjakarta', 2, 'jakarta selatan', 'pancoran', 'kalibata', -6.2625, 106.8290, TRUE),
('TJ-MMPG', 'Mampang Prapatan', 'mampang-prapatan', 'transjakarta', 1, 'jakarta selatan', 'mampang prapatan', 'mampang prapatan', -6.2435, 106.8255, TRUE),
('TJ-KNGN', 'Kuningan', 'kuningan', 'transjakarta', 1, 'jakarta selatan', 'setiabudi', 'karet kuningan', -6.2305, 106.8298, TRUE),
('TJ-KWTI', 'Kwitang', 'kwitang', 'transjakarta', 1, 'jakarta pusat', 'senen', 'kwitang', -6.1795, 106.8395, TRUE),
('TJ-SNEN', 'Pasar Senen', 'pasar-senen', 'transjakarta', 1, 'jakarta pusat', 'senen', 'senen', -6.1748, 106.8432, TRUE),
('TJ-PLGD', 'Pulo Gadung', 'pulo-gadung', 'transjakarta', 2, 'jakarta timur', 'pulo gadung', 'pulo gadung', -6.1885, 106.9035, TRUE),

-- LRT Jabodebek (Light Rail Transit)
('LRT-DKA', 'Dukuh Atas LRT', 'dukuh-atas-lrt', 'lrt', 1, 'jakarta selatan', 'setiabudi', 'setiabudi', -6.2091, 106.8285, TRUE),
('LRT-CWG', 'Cawang', 'cawang', 'lrt', 2, 'jakarta timur', 'jatinegara', 'cawang', -6.2475, 106.8647, TRUE),
('LRT-JAT', 'Jatimulya', 'jatimulya', 'lrt', 4, 'kota bekasi', 'bekasi timur', 'jatimulya', -6.2625, 107.0175, TRUE),
('LRT-HAR', 'Harjamukti', 'harjamukti', 'lrt', 4, 'kota depok', 'cimanggis', 'harjamukti', -6.3705, 106.8970, TRUE),

-- JakLingko Mikrotrans (24 Feeder Routes - Rp 0 Integrated Fare)
('MIK-JAK01', 'Tanjung Priok - Pluit', 'tanjung-priok-pluit', 'mikrotrans', 1, 'jakarta utara', 'tanjung priok', 'tanjung priok', -6.1105, 106.8833, TRUE),
('MIK-JAK02', 'Kampung Rambutan - Duren Sawit', 'kampung-rambutan-duren-sawit', 'mikrotrans', 1, 'jakarta timur', 'ciracas', 'kampung rambutan', -6.3108, 106.8804, TRUE),
('MIK-JAK03', 'Lebak Bulus - Andara', 'lebak-bulus-andara', 'mikrotrans', 1, 'jakarta selatan', 'cilandak', 'lebak bulus', -6.2890, 106.7740, TRUE),
('MIK-JAK05', 'Semper - Rorotan', 'semper-rorotan', 'mikrotrans', 1, 'jakarta utara', 'cilincing', 'semper barat', -6.1289, 106.9150, TRUE),
('MIK-JAK06', 'Kampung Rambutan - Pondok Gede', 'kampung-rambutan-pondok-gede', 'mikrotrans', 1, 'jakarta timur', 'ciracas', 'kampung rambutan', -6.3105, 106.8812, TRUE),
('MIK-JAK07', 'Tanah Abang - Grogol via Tawakal', 'tanah-abang-grogol-via-tawakal', 'mikrotrans', 1, 'jakarta pusat', 'tanah abang', 'kampung bali', -6.1852, 106.8140, TRUE),
('MIK-JAK08', 'Roxy - Bendungan Hilir', 'roxy-bendungan-hilir', 'mikrotrans', 1, 'jakarta pusat', 'gambir', 'cideng', -6.1712, 106.8055, TRUE),
('MIK-JAK09', 'Roxy Mas - Karet Tengsin', 'roxy-mas-karet-tengsin', 'mikrotrans', 1, 'jakarta pusat', 'tanah abang', 'karet tengsin', -6.2030, 106.8180, TRUE),
('MIK-JAK10', 'Tanah Abang - Kota', 'tanah-abang-kota', 'mikrotrans', 1, 'jakarta pusat', 'tanah abang', 'kampung bali', -6.1860, 106.8155, TRUE),
('MIK-JAK11', 'Tanah Abang - Kebayoran Lama', 'tanah-abang-kebayoran-lama', 'mikrotrans', 1, 'jakarta selatan', 'kebayoran lama', 'kebayoran lama utara', -6.2380, 106.7760, TRUE),
('MIK-JAK12', 'Tanah Abang - Pos Pengumben', 'tanah-abang-pos-pengumben', 'mikrotrans', 1, 'jakarta pusat', 'tanah abang', 'kebon kacang', -6.1895, 106.8170, TRUE),
('MIK-JAK13', 'Tanah Abang - Kota Intan', 'tanah-abang-kota-intan', 'mikrotrans', 1, 'jakarta barat', 'tambora', 'roa malaka', -6.1360, 106.8090, TRUE),
('MIK-JAK14', 'Tanah Abang - Meruya', 'tanah-abang-meruya', 'mikrotrans', 1, 'jakarta pusat', 'tanah abang', 'petamburan', -6.1950, 106.8080, TRUE),
('MIK-JAK15', 'Bulak Turi - Tanjung Priok', 'bulak-turi-tanjung-priok', 'mikrotrans', 1, 'jakarta utara', 'tanjung priok', 'kebon bawang', -6.1180, 106.8920, TRUE),
('MIK-JAK16', 'Cililitan - Condet', 'cililitan-condet', 'mikrotrans', 1, 'jakarta timur', 'kramat jati', 'cililitan', -6.2650, 106.8680, TRUE),
('MIK-JAK19', 'Pinang Ranti - Setu', 'pinang-ranti-setu', 'mikrotrans', 1, 'jakarta timur', 'makasar', 'pinang ranti', -6.2910, 106.8910, TRUE),
('MIK-JAK24', 'Senen - Pulo Gadung via Salemba', 'senen-pulo-gadung-via-salemba', 'mikrotrans', 1, 'jakarta pusat', 'senen', 'senen', -6.1750, 106.8430, TRUE),
('MIK-JAK27', 'Pulo Gebang - Rorotan', 'pulo-gebang-rorotan', 'mikrotrans', 1, 'jakarta timur', 'cakung', 'pulo gebang', -6.2120, 106.9530, TRUE),
('MIK-JAK30', 'Grogol - Meruya via Roxy Mas', 'grogol-meruya-via-roxy-mas', 'mikrotrans', 1, 'jakarta barat', 'grogol petamburan', 'grogol', -6.1670, 106.7900, TRUE),
('MIK-JAK31', 'Blok M - Pondok Labu', 'blok-m-pondok-labu', 'mikrotrans', 1, 'jakarta selatan', 'cilandak', 'pondok labu', -6.3050, 106.7980, TRUE),
('MIK-JAK32', 'Petukangan Kidul - Lebak Bulus', 'petukangan-kidul-lebak-bulus', 'mikrotrans', 1, 'jakarta selatan', 'pesanggrahan', 'petukangan kidul', -6.2350, 106.7580, TRUE),
('MIK-JAK33', 'Pulo Gadung - Kota via Mangga Dua', 'pulo-gadung-kota-via-mangga-dua', 'mikrotrans', 1, 'jakarta timur', 'pulo gadung', 'pulo gadung', -6.1880, 106.9030, TRUE),
('MIK-JAK45', 'Lebak Bulus - Ragunan', 'lebak-bulus-ragunan', 'mikrotrans', 1, 'jakarta selatan', 'pasar minggu', 'ragunan', -6.3080, 106.8190, TRUE),
('MIK-JAK48A', 'Stasiun Tebet - Rusun Karet Tengsin', 'stasiun-tebet-rusun-karet-tengsin', 'mikrotrans', 1, 'jakarta selatan', 'tebet', 'tebet barat', -6.2260, 106.8580, TRUE)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    mode = EXCLUDED.mode,
    zone_index = EXCLUDED.zone_index,
    city = EXCLUDED.city,
    district = EXCLUDED.district,
    subdistrict = EXCLUDED.subdistrict,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    is_active = EXCLUDED.is_active;

-- Transit Routes & Route Stops Network
INSERT INTO routes (id, code, name, slug, mode, color_hex, is_active) VALUES
('MRT-M1', 'M1', 'MRT North-South Line (Lebak Bulus - Bundaran HI)', 'mrt-north-south-line-lebak-bulus-bundaran-hi', 'mrt', '#1e439b', TRUE),
('TJ-1', '1', 'Koridor 1 (Blok M - Kota)', 'koridor-1-blok-m-kota', 'transjakarta', '#e30613', TRUE),
('TJ-2', '2', 'Koridor 2 (Monas - Pulo Gadung)', 'koridor-2-monas-pulo-gadung', 'transjakarta', '#0066b2', TRUE),
('TJ-6', '6', 'Koridor 6 (Ragunan - Galunggung)', 'koridor-6-ragunan-galunggung', 'transjakarta', '#008040', TRUE),
('LRT-BK', 'BK', 'LRT Bekasi Line (Dukuh Atas - Jatimulya)', 'lrt-bekasi-line-dukuh-atas-jatimulya', 'lrt', '#d4145a', TRUE),
('LRT-CB', 'CB', 'LRT Cibubur Line (Dukuh Atas - Harjamukti)', 'lrt-cibubur-line-dukuh-atas-harjamukti', 'lrt', '#f39200', TRUE),
('MIK-JAK01', 'JAK01', 'Tanjung Priok - Pluit', 'tanjung-priok-pluit', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK02', 'JAK02', 'Kampung Rambutan - Duren Sawit', 'kampung-rambutan-duren-sawit', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK03', 'JAK03', 'Lebak Bulus - Andara', 'lebak-bulus-andara', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK05', 'JAK05', 'Semper - Rorotan', 'semper-rorotan', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK06', 'JAK06', 'Kampung Rambutan - Pondok Gede', 'kampung-rambutan-pondok-gede', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK07', 'JAK07', 'Tanah Abang - Grogol via Tawakal', 'tanah-abang-grogol-via-tawakal', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK08', 'JAK08', 'Roxy - Bendungan Hilir', 'roxy-bendungan-hilir', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK09', 'JAK09', 'Roxy Mas - Karet Tengsin', 'roxy-mas-karet-tengsin', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK10', 'JAK10', 'Tanah Abang - Kota', 'tanah-abang-kota', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK11', 'JAK11', 'Tanah Abang - Kebayoran Lama', 'tanah-abang-kebayoran-lama', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK12', 'JAK12', 'Tanah Abang - Pos Pengumben', 'tanah-abang-pos-pengumben', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK13', 'JAK13', 'Tanah Abang - Kota Intan', 'tanah-abang-kota-intan', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK14', 'JAK14', 'Tanah Abang - Meruya', 'tanah-abang-meruya', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK15', 'JAK15', 'Bulak Turi - Tanjung Priok', 'bulak-turi-tanjung-priok', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK16', 'JAK16', 'Cililitan - Condet', 'cililitan-condet', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK19', 'JAK19', 'Pinang Ranti - Setu', 'pinang-ranti-setu', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK24', 'JAK24', 'Senen - Pulo Gadung via Salemba', 'senen-pulo-gadung-via-salemba', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK27', 'JAK27', 'Pulo Gebang - Rorotan', 'pulo-gebang-rorotan', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK30', 'JAK30', 'Grogol - Meruya via Roxy Mas', 'grogol-meruya-via-roxy-mas', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK31', 'JAK31', 'Blok M - Pondok Labu', 'blok-m-pondok-labu', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK32', 'JAK32', 'Petukangan Kidul - Lebak Bulus', 'petukangan-kidul-lebak-bulus', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK33', 'JAK33', 'Pulo Gadung - Kota via Mangga Dua', 'pulo-gadung-kota-via-mangga-dua', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK45', 'JAK45', 'Lebak Bulus - Ragunan', 'lebak-bulus-ragunan', 'mikrotrans', '#0096d6', TRUE),
('MIK-JAK48A', 'JAK48A', 'Stasiun Tebet - Rusun Karet Tengsin', 'stasiun-tebet-rusun-karet-tengsin', 'mikrotrans', '#0096d6', TRUE)
ON CONFLICT (id) DO UPDATE SET
    code = EXCLUDED.code,
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    mode = EXCLUDED.mode,
    color_hex = EXCLUDED.color_hex,
    is_active = EXCLUDED.is_active;

INSERT INTO route_stations (route_id, station_id, stop_sequence, distance_from_origin_km) VALUES
-- MRT North-South Line (Lebak Bulus - Bundaran HI)
('MRT-M1', 'MRT-BLB', 1, 0.0),
('MRT-M1', 'MRT-BLM', 2, 6.0),
('MRT-M1', 'MRT-DKA', 3, 11.2),
('MRT-M1', 'MRT-BHI', 4, 15.7),

-- TransJakarta Corridor 1 (Blok M - Kota)
('TJ-1', 'TJ-BLKM', 1, 0.0),
('TJ-1', 'TJ-CSW', 2, 1.2),
('TJ-1', 'TJ-BNSN', 3, 2.8),
('TJ-1', 'TJ-GBK', 4, 4.1),
('TJ-1', 'TJ-PLDA', 5, 5.2),
('TJ-1', 'TJ-BDHL', 6, 6.5),
('TJ-1', 'TJ-KART', 7, 7.2),
('TJ-1', 'TJ-TOSR', 8, 8.2),
('TJ-1', 'TJ-BNHI', 9, 9.0),
('TJ-1', 'TJ-SRNH', 10, 9.8),
('TJ-1', 'TJ-BKID', 11, 10.6),
('TJ-1', 'TJ-MNAS', 12, 11.5),
('TJ-1', 'TJ-HRMN', 13, 13.0),
('TJ-1', 'TJ-MGBS', 14, 14.2),
('TJ-1', 'TJ-GLDK', 15, 15.0),
('TJ-1', 'TJ-KOTA', 16, 16.2),

-- TransJakarta Corridor 6 (Ragunan - Galunggung/Tosari)
('TJ-6', 'TJ-RAGN', 1, 0.0),
('TJ-6', 'TJ-JATI', 2, 2.2),
('TJ-6', 'TJ-PJT', 3, 4.1),
('TJ-6', 'TJ-BUNC', 4, 6.0),
('TJ-6', 'TJ-MMPG', 5, 8.2),
('TJ-6', 'TJ-KNGN', 6, 10.5),
('TJ-6', 'TJ-TOSR', 7, 14.0),

-- TransJakarta Corridor 2 (Monas - Pulo Gadung)
('TJ-2', 'TJ-MNAS', 1, 0.0),
('TJ-2', 'TJ-KWTI', 2, 1.8),
('TJ-2', 'TJ-SNEN', 3, 3.2),
('TJ-2', 'TJ-PLGD', 4, 12.5),

-- LRT Jabodebek Bekasi Line
('LRT-BK', 'LRT-DKA', 1, 0.0),
('LRT-BK', 'LRT-CWG', 2, 8.5),
('LRT-BK', 'LRT-JAT', 3, 27.3),

-- LRT Jabodebek Cibubur Line
('LRT-CB', 'LRT-DKA', 1, 0.0),
('LRT-CB', 'LRT-CWG', 2, 8.5),
('LRT-CB', 'LRT-HAR', 3, 24.3),

-- Mikrotrans Single-Route Bindings
('MIK-JAK01', 'MIK-JAK01', 1, 0.0),
('MIK-JAK02', 'MIK-JAK02', 1, 0.0),
('MIK-JAK03', 'MIK-JAK03', 1, 0.0),
('MIK-JAK05', 'MIK-JAK05', 1, 0.0),
('MIK-JAK06', 'MIK-JAK06', 1, 0.0),
('MIK-JAK07', 'MIK-JAK07', 1, 0.0),
('MIK-JAK08', 'MIK-JAK08', 1, 0.0),
('MIK-JAK09', 'MIK-JAK09', 1, 0.0),
('MIK-JAK10', 'MIK-JAK10', 1, 0.0),
('MIK-JAK11', 'MIK-JAK11', 1, 0.0),
('MIK-JAK12', 'MIK-JAK12', 1, 0.0),
('MIK-JAK13', 'MIK-JAK13', 1, 0.0),
('MIK-JAK14', 'MIK-JAK14', 1, 0.0),
('MIK-JAK15', 'MIK-JAK15', 1, 0.0),
('MIK-JAK16', 'MIK-JAK16', 1, 0.0),
('MIK-JAK19', 'MIK-JAK19', 1, 0.0),
('MIK-JAK24', 'MIK-JAK24', 1, 0.0),
('MIK-JAK27', 'MIK-JAK27', 1, 0.0),
('MIK-JAK30', 'MIK-JAK30', 1, 0.0),
('MIK-JAK31', 'MIK-JAK31', 1, 0.0),
('MIK-JAK32', 'MIK-JAK32', 1, 0.0),
('MIK-JAK33', 'MIK-JAK33', 1, 0.0),
('MIK-JAK45', 'MIK-JAK45', 1, 0.0),
('MIK-JAK48A', 'MIK-JAK48A', 1, 0.0)
ON CONFLICT (route_id, station_id) DO UPDATE SET
    stop_sequence = EXCLUDED.stop_sequence,
    distance_from_origin_km = EXCLUDED.distance_from_origin_km;

-- Transit Smart Cards (Real-World 16-Digit PANs)
INSERT INTO cards (id, card_number, balance, status, created_at, updated_at) VALUES
-- Mandiri e-Money (JakLingko Edition) - General Sample & Happy Path
('98293af8-5358-4f81-bd6a-b99b3a759575', '6032981012345678', 100000, 'active', NOW() - INTERVAL '30 days', NOW()),

-- BCA Flazz (JakLingko Edition) - Exact Minimum Balance Boundary (Rp 5,000)
('7377fb10-f01e-418b-bf80-eb0b9ab94ff7', '6008182055667788', 5000, 'active', NOW() - INTERVAL '14 days', NOW()),

-- BNI TapCash (JakLingko Edition) - Insufficient Balance Test (Rp 2,500 < Rp 5,000)
('82757ad8-ace4-43de-bc75-1b8ac86e9d9f', '7546123099887766', 2500, 'active', NOW() - INTERVAL '7 days', NOW()),

-- BRI Brizzi (JakLingko Edition) - Zero Balance Test (Rp 0)
('846d236b-a375-4c00-a9c3-7fc4ef57ef89', '6013014011223344', 0, 'active', NOW() - INTERVAL '3 days', NOW()),

-- Bank DKI JakCard - Active In-Progress Journey (Tapped in at MRT Dukuh Atas)
('a3e48f9d-fc17-4b8e-9611-c200f2208b51', '6036005088776655', 75000, 'active', NOW() - INTERVAL '1 day', NOW()),

-- JakLingko Multi-Trip Mobility Pass - Expired Journey Test (Tapped in > 4h ago)
('60a68cb7-a6d9-4735-b849-5408033183ac', '6032981144332211', 60000, 'active', NOW() - INTERVAL '2 days', NOW()),

-- JakLingko Tourist Pass - Turnaround Grace Window (< 3m exit cancel test)
('8f169122-5c33-4317-83d9-94f6ef3005be', '6008187012349876', 50000, 'active', NOW() - INTERVAL '1 day', NOW()),

-- BNI TapCash - Suspended Account Status
('b456c3d4-08ab-4baf-8d8d-fd3c81f11541', '7546129033445566', 25000, 'suspended', NOW() - INTERVAL '5 days', NOW()),

-- Bank DKI JakCard - Blocked / Stolen Status
('6da952a7-9e93-4503-a4c3-3ffcb12857d3', '6036008066554433', 50000, 'blocked', NOW() - INTERVAL '10 days', NOW())
ON CONFLICT (card_number) DO UPDATE SET
    balance = EXCLUDED.balance,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Transit Journeys (Completed History & Active States)
INSERT INTO journeys (id, card_id, entry_station_id, exit_station_id, entry_time, exit_time, fare_amount, status, created_at) VALUES
-- Completed Journey 1 for Card 1 (MRT Lebak Bulus -> Dukuh Atas BNI)
('7c92a103-73b6-49ab-b79f-c46682de7e84', '98293af8-5358-4f81-bd6a-b99b3a759575', 'MRT-BLB', 'MRT-DKA', 
 NOW() - INTERVAL '2 hours', NOW() - INTERVAL '90 minutes', 14000, 'completed', NOW() - INTERVAL '2 hours'),

-- Completed Journey 2 for Card 1 (TransJakarta Tosari -> Monas under JakLingko 180m transfer discount)
('a4ef9af5-127c-4458-8700-e6e2e22123eb', '98293af8-5358-4f81-bd6a-b99b3a759575', 'TJ-TOSR', 'TJ-MNAS', 
 NOW() - INTERVAL '75 minutes', NOW() - INTERVAL '45 minutes', 3500, 'completed', NOW() - INTERVAL '75 minutes'),

-- Active In-Progress Journey for Card 5 (Tapped in at Dukuh Atas BNI 15m ago)
('dfcab922-e978-4397-911e-53efd31d2b88', 'a3e48f9d-fc17-4b8e-9611-c200f2208b51', 'MRT-DKA', NULL, 
 NOW() - INTERVAL '15 minutes', NULL, 0, 'in_progress', NOW() - INTERVAL '15 minutes'),

-- Expired Journey for Card 6 (Tapped in at Kota 5 hours ago - eligible for sweeper penalty)
('283a947c-77cf-4f88-ab04-4bcfe6f8896c', '60a68cb7-a6d9-4735-b849-5408033183ac', 'TJ-KOTA', NULL, 
 NOW() - INTERVAL '5 hours', NULL, 0, 'in_progress', NOW() - INTERVAL '5 hours'),

-- Turnaround Grace Window Journey for Card 7 (Tapped in at Bundaran HI 1m ago - < 3m grace)
('20aecb4e-3dcf-4d75-9967-fd4edd1d8cac', '8f169122-5c33-4317-83d9-94f6ef3005be', 'MRT-BHI', NULL, 
 NOW() - INTERVAL '1 minute', NULL, 0, 'in_progress', NOW() - INTERVAL '1 minute')
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    fare_amount = EXCLUDED.fare_amount,
    exit_station_id = EXCLUDED.exit_station_id,
    exit_time = EXCLUDED.exit_time;

-- Ledger Transactions (Audit Trail for Card 1)
INSERT INTO ledger_transactions (id, card_id, journey_id, transaction_type, amount, balance_after, idempotency_key, created_at) VALUES
-- Initial Card Provisioning Top-Up (+Rp 100,000)
('97bde859-0c66-4c19-9e79-4e851b74447c', '98293af8-5358-4f81-bd6a-b99b3a759575', NULL, 
 'top_up', 100000, 100000, 'seed-topup-card1-init', NOW() - INTERVAL '3 hours'),

-- Fare Deduction for Journey 1 (-Rp 14,000)
('5ac6a9c9-e55f-43d1-a3c3-fe48795f5bc9', '98293af8-5358-4f81-bd6a-b99b3a759575', '7c92a103-73b6-49ab-b79f-c46682de7e84', 
 'fare_deduction', -14000, 86000, 'seed-fare-card1-leg1', NOW() - INTERVAL '90 minutes'),

-- Fare Deduction for Journey 2 (-Rp 3,500 under JakLingko transfer cap)
('02a446fa-7c12-4ec8-b2e3-ea7638fdf49c', '98293af8-5358-4f81-bd6a-b99b3a759575', 'a4ef9af5-127c-4458-8700-e6e2e22123eb', 
 'fare_deduction', -3500, 82500, 'seed-fare-card1-leg2', NOW() - INTERVAL '45 minutes'),

-- Top-Up to restore clean round balance (+Rp 17,500 -> Rp 100,000)
('070dbfa7-a6bf-469a-a545-3368ea5dafc1', '98293af8-5358-4f81-bd6a-b99b3a759575', NULL, 
 'top_up', 17500, 100000, 'seed-topup-card1-restore', NOW() - INTERVAL '30 minutes')
ON CONFLICT (id) DO NOTHING;
