import { StatusBar } from 'expo-status-bar';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Modal,
  Platform,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';

// ─── Change this to your Mac's local IP (run `ipconfig getifaddr en0`) ────────
const API_BASE = 'http://192.168.100.236:8080';
const POLL_INTERVAL_MS = 10_000;

// ─── colours ──────────────────────────────────────────────────────────────────
const DARK = '#0d1117';
const CARD = '#161b22';
const BORDER = '#30363d';
const GREEN = '#3fb950';
const RED = '#f85149';
const YELLOW = '#d29922';
const ACCENT = '#58a6ff';
const TEXT = '#e6edf3';
const SUBTEXT = '#8b949e';

const DECISION_COLOR = { BUY: GREEN, SELL: RED, HOLD: YELLOW };
const CONFIDENCE_COLOR = { High: GREEN, Medium: YELLOW, Low: RED };

const fmt = (n, decimals = 2) =>
  Number(n).toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });

// ─── API helpers ──────────────────────────────────────────────────────────────
async function apiFetch(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  if (res.status === 204) return null;
  return res.json();
}

const fetchPortfolio = () => apiFetch('/portfolio');
const postOrder = (body) => apiFetch('/orders', { method: 'POST', body: JSON.stringify(body) });
const deleteStop = (id) => apiFetch(`/stops/${id}`, { method: 'DELETE' });
const executeStop = (id) => apiFetch(`/stops/${id}/execute`, { method: 'POST' });

// ─── App ──────────────────────────────────────────────────────────────────────
export default function App() {
  const [tab, setTab] = useState('portfolio');
  const [holdings, setHoldings] = useState([]);
  const [stops, setStops] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [connected, setConnected] = useState(null); // null=unknown, true, false
  const [lastUpdated, setLastUpdated] = useState(null);

  // Order modal
  const [orderModal, setOrderModal] = useState(false);
  const [side, setSide] = useState('BUY');
  const [oTicker, setOTicker] = useState('');
  const [oQty, setOQty] = useState('');
  const [oPrice, setOPrice] = useState('');
  const [oSearch, setOSearch] = useState('');
  const [oDropdown, setODropdown] = useState(false);

  // Stop modal
  const [stopModal, setStopModal] = useState(false);
  const [sTicker, setSTicker] = useState('');
  const [sQty, setSQty] = useState('');
  const [sPrice, setSPrice] = useState('');
  const [sSearch, setSSearch] = useState('');
  const [sDropdown, setSDrop] = useState(false);

  const pollRef = useRef(null);

  // ── fetch ──────────────────────────────────────────────────────────────────
  const load = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const data = await fetchPortfolio();
      setHoldings(data.holdings || []);
      setStops(data.stops || []);
      setConnected(true);
      setLastUpdated(new Date());
    } catch {
      setConnected(false);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    load();
    pollRef.current = setInterval(() => load(true), POLL_INTERVAL_MS);
    return () => clearInterval(pollRef.current);
  }, [load]);

  const onRefresh = () => { setRefreshing(true); load(); };

  // ── order submit ──────────────────────────────────────────────────────────
  const submitOrder = async () => {
    const qty = parseFloat(oQty);
    const price = parseFloat(oPrice);
    if (!oTicker) return Alert.alert('Error', 'Select a ticker.');
    if (!qty || qty <= 0) return Alert.alert('Error', 'Enter a valid quantity.');
    if (!price || price <= 0) return Alert.alert('Error', 'Enter a valid price.');
    try {
      await postOrder({ type: side.toLowerCase(), ticker: oTicker, qty, price });
      setOrderModal(false);
      load(true);
    } catch (e) {
      Alert.alert('Order failed', e.message);
    }
  };

  // ── stop submit ───────────────────────────────────────────────────────────
  const submitStop = async () => {
    const qty = parseFloat(sQty);
    const price = parseFloat(sPrice);
    if (!sTicker) return Alert.alert('Error', 'Select a ticker.');
    if (!qty || qty <= 0) return Alert.alert('Error', 'Enter a valid quantity.');
    if (!price || price <= 0) return Alert.alert('Error', 'Enter a valid stop price.');
    try {
      await postOrder({ type: 'stop_loss', ticker: sTicker, qty, price });
      setStopModal(false);
      load(true);
    } catch (e) {
      Alert.alert('Failed', e.message);
    }
  };

  // ── stop actions ──────────────────────────────────────────────────────────
  const onExecuteStop = (s) => {
    Alert.alert(
      'Execute Stop Loss',
      `Sell ${fmt(s.qty, 4)} ${s.ticker} @ stop $${fmt(s.stop_price)}?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Execute', style: 'destructive', onPress: async () => {
            try { await executeStop(s.id); load(true); }
            catch (e) { Alert.alert('Error', e.message); }
          }
        },
      ]
    );
  };

  const onDeleteStop = (s) => {
    Alert.alert('Delete Stop Loss', 'Remove this stop-loss order?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete', style: 'destructive', onPress: async () => {
          try { await deleteStop(s.id); load(true); }
          catch (e) { Alert.alert('Error', e.message); }
        }
      },
    ]);
  };

  // ── derived ───────────────────────────────────────────────────────────────
  const positions = holdings.filter((h) => h.qty > 0);
  const watchlist = holdings.filter((h) => h.qty === 0);
  const allTickers = holdings.map((h) => h.ticker);
  const heldTickers = positions.map((h) => h.ticker);

  const openOrder = () => {
    setSide('BUY'); setOTicker(''); setOQty(''); setOPrice('');
    setOSearch(''); setODropdown(false); setOrderModal(true);
  };
  const openStop = () => {
    setSTicker(''); setSQty(''); setSPrice('');
    setSSearch(''); setSDrop(false); setStopModal(true);
  };

  // ── render ────────────────────────────────────────────────────────────────
  return (
    <View style={styles.root}>
      <StatusBar style="light" />

      {/* Header */}
      <View style={styles.header}>
        <View>
          <Text style={styles.headerTitle}>Jobot</Text>
          {lastUpdated && (
            <Text style={styles.headerSub}>
              <Text style={{ color: connected ? GREEN : connected === false ? RED : SUBTEXT }}>● </Text>
              <Text>{connected ? 'Live' : 'Offline'} · {lastUpdated.toLocaleTimeString()}</Text>
            </Text>
          )}
        </View>
        {tab === 'portfolio' ? (
          <TouchableOpacity style={styles.headerBtn} onPress={openOrder}>
            <Text style={styles.headerBtnText}>+ Order</Text>
          </TouchableOpacity>
        ) : (
          <TouchableOpacity style={[styles.headerBtn, { backgroundColor: RED }]} onPress={openStop}>
            <Text style={styles.headerBtnText}>+ Stop Loss</Text>
          </TouchableOpacity>
        )}
      </View>

      {/* Tab bar */}
      <View style={styles.tabBar}>
        {[
          { key: 'portfolio', label: 'Portfolio' },
          { key: 'stops', label: `Stop Loss${stops.length ? ` (${stops.length})` : ''}` },
        ].map(({ key, label }) => (
          <TouchableOpacity
            key={key}
            style={[styles.tabItem, tab === key && styles.tabActive]}
            onPress={() => setTab(key)}
          >
            <Text style={[styles.tabText, tab === key && styles.tabTextActive]}>{label}</Text>
          </TouchableOpacity>
        ))}
      </View>

      {/* Connection banner */}
      {connected === false && (
        <View style={styles.offlineBanner}>
          <Text style={styles.offlineText}>Cannot reach backend · {API_BASE}</Text>
        </View>
      )}

      {/* Loading spinner */}
      {loading && holdings.length === 0 && (
        <View style={styles.centered}>
          <ActivityIndicator color={ACCENT} size="large" />
          <Text style={[styles.cardSub, { marginTop: 12 }]}>Connecting to backend…</Text>
        </View>
      )}

      {/* Portfolio tab */}
      {tab === 'portfolio' && !loading && (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={ACCENT} />}
        >
          <PortfolioSummary holdings={positions} />
          <Text style={styles.sectionLabel}>POSITIONS</Text>
          {positions.map((h) => <HoldingCard key={h.ticker} h={h} stops={stops} />)}

          <Text style={[styles.sectionLabel, { marginTop: 24 }]}>WATCHLIST</Text>
          <View style={styles.watchRow}>
            {watchlist.map((h) => (
              <View key={h.ticker} style={styles.watchChip}>
                <Text style={styles.watchText}>{h.ticker}</Text>
                {h.latest_prediction && (
                  <Text style={[styles.watchDecision, { color: DECISION_COLOR[h.latest_prediction.decision] || SUBTEXT }]}>
                    {h.latest_prediction.decision}
                  </Text>
                )}
              </View>
            ))}
          </View>
        </ScrollView>
      )}

      {/* Stop Loss tab */}
      {tab === 'stops' && !loading && (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={ACCENT} />}
        >
          {stops.length === 0 ? (
            <View style={styles.empty}>
              <Text style={styles.emptyText}>No stop-loss orders.</Text>
              <Text style={styles.emptyHint}>Tap "+ Stop Loss" to add one.</Text>
            </View>
          ) : stops.map((s) => (
            <View key={s.id} style={styles.stopCard}>
              <View style={styles.cardRow}>
                <Text style={styles.cardTicker}>{s.ticker}</Text>
                <View style={styles.stopPriceBadge}>
                  <Text style={styles.stopPriceText}>Stop @ ${fmt(s.stop_price)}</Text>
                </View>
              </View>
              <View style={styles.cardRow}>
                <Text style={styles.cardSub}>Qty to sell</Text>
                <Text style={styles.cardSub}>{fmt(s.qty, 4)} shares</Text>
              </View>
              <Text style={[styles.cardSub, { marginTop: 2 }]}>
                Set {new Date(s.created_at).toLocaleDateString()}
              </Text>
              <View style={[styles.cardRow, { marginTop: 10, gap: 8 }]}>
                <TouchableOpacity style={styles.executeBtn} onPress={() => onExecuteStop(s)}>
                  <Text style={styles.executeBtnText}>Execute</Text>
                </TouchableOpacity>
                <TouchableOpacity style={styles.deleteBtn} onPress={() => onDeleteStop(s)}>
                  <Text style={styles.deleteBtnText}>Delete</Text>
                </TouchableOpacity>
              </View>
            </View>
          ))}
        </ScrollView>
      )}

      {/* ══ ORDER MODAL ══ */}
      <Modal visible={orderModal} animationType="slide" transparent>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.overlay}>
          <View style={styles.sheet}>
            <Text style={styles.sheetTitle}>New Order</Text>
            <View style={styles.toggle}>
              {['BUY', 'SELL'].map((s) => (
                <TouchableOpacity
                  key={s}
                  style={[styles.toggleBtn, side === s && (s === 'BUY' ? styles.toggleBuy : styles.toggleSell)]}
                  onPress={() => setSide(s)}
                >
                  <Text style={[styles.toggleText, side === s && styles.toggleTextActive]}>{s}</Text>
                </TouchableOpacity>
              ))}
            </View>
            <TickerPicker
              label="Ticker"
              tickers={allTickers.filter((t) => t.toLowerCase().includes(oSearch.toLowerCase()))}
              value={oTicker} search={oSearch} onSearch={setOSearch}
              open={oDropdown} setOpen={setODropdown} onSelect={setOTicker}
            />
            <FieldLabel>Quantity</FieldLabel>
            <TextInput style={styles.input} placeholder="e.g. 2.5" placeholderTextColor="#888" keyboardType="decimal-pad" value={oQty} onChangeText={setOQty} />
            <FieldLabel>Price per share ($)</FieldLabel>
            <TextInput style={styles.input} placeholder="e.g. 185.00" placeholderTextColor="#888" keyboardType="decimal-pad" value={oPrice} onChangeText={setOPrice} />
            <View style={styles.actions}>
              <TouchableOpacity style={styles.cancelBtn} onPress={() => setOrderModal(false)}>
                <Text style={styles.cancelText}>Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity style={[styles.submitBtn, side === 'SELL' && { backgroundColor: RED }]} onPress={submitOrder}>
                <Text style={styles.submitText}>{side}</Text>
              </TouchableOpacity>
            </View>
          </View>
        </KeyboardAvoidingView>
      </Modal>

      {/* ══ STOP LOSS MODAL ══ */}
      <Modal visible={stopModal} animationType="slide" transparent>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.overlay}>
          <View style={styles.sheet}>
            <Text style={styles.sheetTitle}>New Stop-Loss Order</Text>
            <Text style={styles.sheetHint}>Shares are sold when you manually execute or the backend triggers the stop.</Text>
            <TickerPicker
              label="Ticker (held positions only)"
              tickers={heldTickers.filter((t) => t.toLowerCase().includes(sSearch.toLowerCase()))}
              value={sTicker} search={sSearch} onSearch={setSSearch}
              open={sDropdown} setOpen={setSDrop} onSelect={setSTicker}
            />
            <FieldLabel>Qty to sell</FieldLabel>
            <TextInput style={styles.input} placeholder="e.g. 2.5" placeholderTextColor="#888" keyboardType="decimal-pad" value={sQty} onChangeText={setSQty} />
            <FieldLabel>Stop price ($)</FieldLabel>
            <TextInput style={styles.input} placeholder="e.g. 170.00" placeholderTextColor="#888" keyboardType="decimal-pad" value={sPrice} onChangeText={setSPrice} />
            <View style={styles.actions}>
              <TouchableOpacity style={styles.cancelBtn} onPress={() => setStopModal(false)}>
                <Text style={styles.cancelText}>Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity style={[styles.submitBtn, { backgroundColor: RED }]} onPress={submitStop}>
                <Text style={styles.submitText}>Set Stop Loss</Text>
              </TouchableOpacity>
            </View>
          </View>
        </KeyboardAvoidingView>
      </Modal>
    </View>
  );
}

// ─── PortfolioSummary bar ─────────────────────────────────────────────────────
function PortfolioSummary({ holdings }) {
  let costBasis = 0, marketValue = 0, priceCount = 0;
  for (const h of holdings) {
    costBasis += h.qty * h.avg_cost;
    if (h.latest_prediction?.price) {
      marketValue += h.qty * h.latest_prediction.price;
      priceCount++;
    }
  }
  const pl    = marketValue - costBasis;
  const plPct = costBasis > 0 ? (pl / costBasis) * 100 : 0;
  const plColor = pl >= 0 ? GREEN : RED;
  const sign  = pl >= 0 ? '+' : '';

  return (
    <View style={styles.summaryBar}>
      <View style={styles.summaryCol}>
        <Text style={styles.summaryLabel}>Cost Basis</Text>
        <Text style={styles.summaryValue}>${fmt(costBasis)}</Text>
      </View>
      <View style={styles.summaryDivider} />
      <View style={styles.summaryCol}>
        <Text style={styles.summaryLabel}>Mkt Value{priceCount < holdings.length ? '*' : ''}</Text>
        <Text style={styles.summaryValue}>${fmt(marketValue)}</Text>
      </View>
      <View style={styles.summaryDivider} />
      <View style={styles.summaryCol}>
        <Text style={styles.summaryLabel}>Unrealized P&L</Text>
        <Text style={[styles.summaryValue, { color: plColor }]}>
          {sign}${fmt(Math.abs(pl))} ({sign}{fmt(Math.abs(plPct))}%)
        </Text>
      </View>
    </View>
  );
}

// ─── HoldingCard ──────────────────────────────────────────────────────────────
function HoldingCard({ h, stops }) {
  const [expanded, setExpanded] = useState(false);
  const p  = h.latest_prediction;
  const myStops = stops.filter((s) => s.ticker === h.ticker);
  const decisionColor = DECISION_COLOR[p?.decision] || SUBTEXT;

  // P&L using last known price from prediction
  const lastPrice   = p?.price ?? null;
  const pl          = lastPrice != null ? (lastPrice - h.avg_cost) * h.qty : null;
  const plPct       = lastPrice != null && h.avg_cost > 0 ? ((lastPrice - h.avg_cost) / h.avg_cost) * 100 : null;
  const plColor     = pl != null ? (pl >= 0 ? GREEN : RED) : SUBTEXT;
  const plSign      = pl != null && pl >= 0 ? '+' : '';

  // Trend badge
  const trend = p?.trend60d ?? null;
  const trendColor = trend != null ? (trend >= 0 ? GREEN : RED) : SUBTEXT;
  const trendLabel = trend != null ? `${trend >= 0 ? '+' : ''}${fmt(trend, 1)}% 60d` : null;

  return (
    <View style={styles.card}>
      {/* Row 1: ticker + trend badge */}
      <View style={styles.cardRow}>
        <Text style={styles.cardTicker}>{h.ticker}</Text>
        <View style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
          {trendLabel && (
            <View style={[styles.trendBadge, { borderColor: trendColor }]}>
              <Text style={[styles.trendText, { color: trendColor }]}>{trendLabel}</Text>
            </View>
          )}
          <Text style={styles.cardQty}>{fmt(h.qty, 4)} sh</Text>
        </View>
      </View>

      {/* Row 2: avg cost + P&L */}
      <View style={styles.cardRow}>
        <Text style={styles.cardSub}>Avg ${fmt(h.avg_cost)} · Basis ${fmt(h.qty * h.avg_cost)}</Text>
        {pl != null && (
          <Text style={[styles.plText, { color: plColor }]}>
            {plSign}${fmt(Math.abs(pl))} ({plSign}{fmt(Math.abs(plPct))}%)
          </Text>
        )}
      </View>

      {/* Prediction block */}
      {p ? (
        <View style={styles.predBlock}>
          {/* Decision + confidence + last price */}
          <View style={styles.cardRow}>
            <View style={[styles.decisionBadge, { backgroundColor: decisionColor + '22', borderColor: decisionColor }]}>
              <Text style={[styles.decisionText, { color: decisionColor }]}>{p.decision}</Text>
            </View>
            <Text style={[styles.confidenceText, { color: CONFIDENCE_COLOR[p.confidence] || SUBTEXT }]}>
              {p.confidence} confidence
            </Text>
            <Text style={styles.cardSub}>${fmt(p.price)}</Text>
          </View>

          {/* Reasoning — collapsible */}
          {p.reasoning ? (
            <>
              <Text style={styles.reasoningText} numberOfLines={expanded ? undefined : 2}>{p.reasoning}</Text>
              <TouchableOpacity onPress={() => setExpanded((v) => !v)}>
                <Text style={styles.expandBtn}>{expanded ? 'Show less ▲' : 'View full reasoning ▼'}</Text>
              </TouchableOpacity>
            </>
          ) : null}

          {/* Price target + AI stop */}
          <View style={[styles.cardRow, { marginTop: 6 }]}>
            {p.priceTarget ? <Text style={styles.targetText}>Target {p.priceTarget}</Text> : null}
            {p.stopLoss    ? <Text style={styles.aiStopText}>AI stop {p.stopLoss}</Text>    : null}
          </View>

          <Text style={styles.predDate}>Predicted at {new Date(p.date).toLocaleString()}</Text>
        </View>
      ) : (
        <Text style={[styles.cardSub, { marginTop: 8 }]}>No prediction yet</Text>
      )}


      {/* Active stop-loss badges */}
      {myStops.length > 0 && (
        <View style={styles.stopBadgeRow}>
          {myStops.map((s) => (
            <View key={s.id} style={styles.stopBadge}>
              <Text style={styles.stopBadgeText}>SL ${fmt(s.stop_price)} × {fmt(s.qty, 4)}</Text>
            </View>
          ))}
        </View>
      )}
    </View>
  );
}

// ─── helpers ──────────────────────────────────────────────────────────────────
function FieldLabel({ children }) {
  return <Text style={styles.inputLabel}>{children}</Text>;
}

function TickerPicker({ label, tickers, value, search, onSearch, open, setOpen, onSelect }) {
  return (
    <>
      <FieldLabel>{label}</FieldLabel>
      <TouchableOpacity style={styles.input} onPress={() => setOpen(!open)}>
        <Text style={value ? styles.inputText : styles.inputPlaceholder}>
          {value || 'Select ticker…'}
        </Text>
      </TouchableOpacity>
      {open && (
        <View style={styles.dropdown}>
          <TextInput style={styles.dropdownSearch} placeholder="Search…" placeholderTextColor="#888" value={search} onChangeText={onSearch} autoCapitalize="characters" />
          <FlatList
            data={tickers} keyExtractor={(t) => t} style={{ maxHeight: 150 }}
            renderItem={({ item }) => (
              <TouchableOpacity style={styles.dropdownItem} onPress={() => { onSelect(item); setOpen(false); onSearch(''); }}>
                <Text style={styles.dropdownItemText}>{item}</Text>
              </TouchableOpacity>
            )}
          />
        </View>
      )}
    </>
  );
}

// ─── styles ───────────────────────────────────────────────────────────────────
const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: DARK },

  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingTop: 60, paddingHorizontal: 16, paddingBottom: 12, borderBottomWidth: 1, borderBottomColor: BORDER },
  headerTitle: { color: TEXT, fontSize: 20, fontWeight: '700' },
  headerSub: { color: SUBTEXT, fontSize: 11, marginTop: 2 },
  headerBtn: { backgroundColor: ACCENT, paddingHorizontal: 14, paddingVertical: 7, borderRadius: 8 },
  headerBtnText: { color: '#000', fontWeight: '700', fontSize: 14 },

  tabBar: { flexDirection: 'row', borderBottomWidth: 1, borderBottomColor: BORDER },
  tabItem: { flex: 1, paddingVertical: 12, alignItems: 'center' },
  tabActive: { borderBottomWidth: 2, borderBottomColor: ACCENT },
  tabText: { color: SUBTEXT, fontSize: 14, fontWeight: '600' },
  tabTextActive: { color: ACCENT },

  offlineBanner: { backgroundColor: RED + '33', paddingVertical: 6, paddingHorizontal: 16 },
  offlineText: { color: RED, fontSize: 12, textAlign: 'center' },

  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  scroll: { padding: 16, paddingBottom: 40 },
  sectionLabel: { color: SUBTEXT, fontSize: 11, fontWeight: '600', letterSpacing: 1, marginBottom: 8 },

  // Portfolio summary bar
  summaryBar: { flexDirection: 'row', backgroundColor: CARD, borderRadius: 10, padding: 14, marginBottom: 16, borderWidth: 1, borderColor: BORDER },
  summaryCol: { flex: 1, alignItems: 'center' },
  summaryDivider: { width: 1, backgroundColor: BORDER, marginVertical: 2 },
  summaryLabel: { color: SUBTEXT, fontSize: 10, fontWeight: '600', letterSpacing: 0.5, marginBottom: 4 },
  summaryValue: { color: TEXT, fontSize: 13, fontWeight: '700' },

  card: { backgroundColor: CARD, borderRadius: 10, padding: 14, marginBottom: 10, borderWidth: 1, borderColor: BORDER },
  stopCard: { backgroundColor: CARD, borderRadius: 10, padding: 14, marginBottom: 10, borderWidth: 1, borderColor: RED + '55' },
  cardRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 },
  cardTicker: { color: TEXT, fontSize: 16, fontWeight: '700' },
  cardQty: { color: ACCENT, fontSize: 14, fontWeight: '600' },
  cardSub: { color: SUBTEXT, fontSize: 13 },

  // P&L + trend
  plText: { fontSize: 13, fontWeight: '700' },
  trendBadge: { borderWidth: 1, borderRadius: 5, paddingHorizontal: 6, paddingVertical: 1 },
  trendText: { fontSize: 11, fontWeight: '700' },

  predBlock: { marginTop: 10, paddingTop: 10, borderTopWidth: 1, borderTopColor: BORDER },
  decisionBadge: { borderWidth: 1, borderRadius: 6, paddingHorizontal: 8, paddingVertical: 2 },
  decisionText: { fontWeight: '800', fontSize: 13 },
  confidenceText: { fontSize: 12, fontWeight: '600' },
  summaryText: { color: TEXT, fontSize: 13, marginTop: 6, lineHeight: 18 },
  reasoningText: { color: SUBTEXT, fontSize: 12, marginTop: 4, lineHeight: 18 },
  expandBtn: { color: ACCENT, fontSize: 12, fontWeight: '600', marginTop: 4 },
  keyRiskRow: { flexDirection: 'row', marginTop: 6, backgroundColor: RED + '11', borderRadius: 6, padding: 6 },
  keyRiskLabel: { color: RED, fontSize: 11, fontWeight: '800' },
  keyRiskText: { color: RED + 'cc', fontSize: 11, flex: 1, lineHeight: 16 },
  targetText: { color: GREEN, fontSize: 12, fontWeight: '600' },
  aiStopText: { color: YELLOW, fontSize: 12, fontWeight: '600' },
  predDate: { color: SUBTEXT + '80', fontSize: 11, marginTop: 6 },

  // Weekly summary
  weeklyBlock: { marginTop: 10, paddingTop: 10, borderTopWidth: 1, borderTopColor: BORDER },
  weeklyTitle: { color: SUBTEXT, fontSize: 12, fontWeight: '700' },
  weeklyDominant: { fontSize: 12, fontWeight: '700' },
  weeklyText: { color: TEXT, fontSize: 12, lineHeight: 17, marginTop: 4 },
  themeRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 4, marginTop: 6 },
  themeBadge: { backgroundColor: ACCENT + '22', borderRadius: 4, paddingHorizontal: 6, paddingVertical: 2 },
  themeText: { color: ACCENT, fontSize: 11, fontWeight: '600' },

  stopBadgeRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 6, marginTop: 8 },
  stopBadge: { backgroundColor: RED + '22', borderWidth: 1, borderColor: RED + '55', borderRadius: 6, paddingHorizontal: 8, paddingVertical: 3 },
  stopBadgeText: { color: RED, fontSize: 11, fontWeight: '600' },

  stopPriceBadge: { backgroundColor: RED + '22', borderWidth: 1, borderColor: RED + '55', borderRadius: 6, paddingHorizontal: 8, paddingVertical: 3 },
  stopPriceText: { color: RED, fontSize: 13, fontWeight: '700' },
  executeBtn: { flex: 1, backgroundColor: YELLOW + '22', borderWidth: 1, borderColor: YELLOW, borderRadius: 8, paddingVertical: 8, alignItems: 'center' },
  executeBtnText: { color: YELLOW, fontWeight: '700', fontSize: 13 },
  deleteBtn: { flex: 1, backgroundColor: RED + '22', borderWidth: 1, borderColor: RED + '55', borderRadius: 8, paddingVertical: 8, alignItems: 'center' },
  deleteBtnText: { color: RED, fontWeight: '700', fontSize: 13 },

  watchRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  watchChip: { backgroundColor: CARD, borderRadius: 6, paddingHorizontal: 10, paddingVertical: 5, borderWidth: 1, borderColor: BORDER, alignItems: 'center' },
  watchText: { color: SUBTEXT, fontSize: 13, fontWeight: '600' },
  watchDecision: { fontSize: 10, fontWeight: '800', marginTop: 2 },

  empty: { alignItems: 'center', marginTop: 80 },
  emptyText: { color: SUBTEXT, fontSize: 16, fontWeight: '600' },
  emptyHint: { color: SUBTEXT + '88', fontSize: 13, marginTop: 6 },

  overlay: { flex: 1, justifyContent: 'flex-end', backgroundColor: 'rgba(0,0,0,0.6)' },
  sheet: { backgroundColor: CARD, borderTopLeftRadius: 20, borderTopRightRadius: 20, padding: 24, paddingBottom: 40, borderTopWidth: 1, borderColor: BORDER },
  sheetTitle: { color: TEXT, fontSize: 18, fontWeight: '700', marginBottom: 6 },
  sheetHint: { color: SUBTEXT, fontSize: 13, marginBottom: 8 },
  toggle: { flexDirection: 'row', marginBottom: 4, borderRadius: 8, overflow: 'hidden', borderWidth: 1, borderColor: BORDER },
  toggleBtn: { flex: 1, paddingVertical: 10, alignItems: 'center', backgroundColor: DARK },
  toggleBuy: { backgroundColor: GREEN },
  toggleSell: { backgroundColor: RED },
  toggleText: { color: SUBTEXT, fontWeight: '700' },
  toggleTextActive: { color: '#fff' },
  inputLabel: { color: SUBTEXT, fontSize: 12, marginBottom: 4, marginTop: 12 },
  input: { backgroundColor: DARK, borderRadius: 8, borderWidth: 1, borderColor: BORDER, paddingHorizontal: 12, paddingVertical: 11, color: TEXT, fontSize: 15 },
  inputText: { color: TEXT, fontSize: 15 },
  inputPlaceholder: { color: '#888', fontSize: 15 },
  dropdown: { backgroundColor: DARK, borderWidth: 1, borderColor: BORDER, borderRadius: 8, marginTop: 4, overflow: 'hidden' },
  dropdownSearch: { borderBottomWidth: 1, borderBottomColor: BORDER, paddingHorizontal: 12, paddingVertical: 8, color: TEXT, fontSize: 14 },
  dropdownItem: { paddingHorizontal: 12, paddingVertical: 10 },
  dropdownItemText: { color: TEXT, fontSize: 14 },
  actions: { flexDirection: 'row', gap: 12, marginTop: 24 },
  cancelBtn: { flex: 1, paddingVertical: 13, borderRadius: 10, alignItems: 'center', borderWidth: 1, borderColor: BORDER },
  cancelText: { color: SUBTEXT, fontWeight: '600' },
  submitBtn: { flex: 2, paddingVertical: 13, borderRadius: 10, alignItems: 'center', backgroundColor: GREEN },
  submitText: { color: '#fff', fontWeight: '700', fontSize: 15 },
});
