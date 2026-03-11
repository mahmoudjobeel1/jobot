import * as Haptics from 'expo-haptics';
import { StatusBar } from 'expo-status-bar';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Animated,
  FlatList,
  KeyboardAvoidingView,
  LayoutAnimation,
  Modal,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  UIManager,
  View,
} from 'react-native';

// Enable LayoutAnimation on Android
if (Platform.OS === 'android') UIManager.setLayoutAnimationEnabledExperimental?.(true);

// ─── Config ───────────────────────────────────────────────────────────────────
const API_BASE = 'http://192.168.100.236:8080';
const POLL_INTERVAL_MS = 10_000;

// ─── Colours ──────────────────────────────────────────────────────────────────
const DARK = '#0d1117';
const CARD = '#161b22';
const CARD2 = '#1c2128';
const BORDER = '#30363d';
const GREEN = '#3fb950';
const RED = '#f85149';
const YELLOW = '#d29922';
const ACCENT = '#58a6ff';
const TEXT = '#e6edf3';
const SUBTEXT = '#8b949e';

const DECISION_COLOR = { BUY: GREEN, SELL: RED, HOLD: YELLOW };
const CONFIDENCE_COLOR = { High: GREEN, Medium: YELLOW, Low: RED };

// ─── Formatters ───────────────────────────────────────────────────────────────
const fmt = (n, decimals = 2) =>
  Number(n).toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals });

// ─── API ──────────────────────────────────────────────────────────────────────
async function apiFetch(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) { const t = await res.text(); throw new Error(t || `HTTP ${res.status}`); }
  if (res.status === 204) return null;
  return res.json();
}
const fetchPortfolio = () => apiFetch('/portfolio');
const postOrder = (body) => apiFetch('/orders', { method: 'POST', body: JSON.stringify(body) });
const apiDeleteStop = (id) => apiFetch(`/stops/${id}`, { method: 'DELETE' });
const apiExecuteStop = (id) => apiFetch(`/stops/${id}/execute`, { method: 'POST' });

// ─── Animated press wrapper ───────────────────────────────────────────────────
function ScalePress({ onPress, style, children }) {
  const scale = useRef(new Animated.Value(1)).current;
  const onIn = () => Animated.spring(scale, { toValue: 0.97, useNativeDriver: true, speed: 40 }).start();
  const onOut = () => Animated.spring(scale, { toValue: 1, useNativeDriver: true, speed: 20 }).start();
  return (
    <Pressable onPress={onPress} onPressIn={onIn} onPressOut={onOut}>
      <Animated.View style={[style, { transform: [{ scale }] }]}>{children}</Animated.View>
    </Pressable>
  );
}

// ─── App ──────────────────────────────────────────────────────────────────────
export default function App() {
  const [tab, setTab] = useState('portfolio');
  const [holdings, setHoldings] = useState([]);
  const [stops, setStops] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [connected, setConnected] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  // Order modal state
  const [orderModal, setOrderModal] = useState(false);
  const [side, setSide] = useState('BUY');
  const [oTicker, setOTicker] = useState('');
  const [oQty, setOQty] = useState('');
  const [oPrice, setOPrice] = useState('');
  const [oSearch, setOSearch] = useState('');
  const [oDropdown, setODropdown] = useState(false);

  // Stop modal state
  const [stopModal, setStopModal] = useState(false);
  const [sTicker, setSTicker] = useState('');
  const [sQty, setSQty] = useState('');
  const [sPrice, setSPrice] = useState('');
  const [sSearch, setSSearch] = useState('');
  const [sDropdown, setSDrop] = useState(false);

  const pollRef = useRef(null);

  // ── Data fetching ────────────────────────────────────────────────────────────
  const load = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const data = await fetchPortfolio();
      setHoldings(data.holdings || []);
      setStops(data.stops || []);
      setConnected(true);
      setLastUpdated(new Date());
    } catch { setConnected(false); }
    finally { setLoading(false); setRefreshing(false); }
  }, []);

  useEffect(() => {
    load();
    pollRef.current = setInterval(() => load(true), POLL_INTERVAL_MS);
    return () => clearInterval(pollRef.current);
  }, [load]);

  const onRefresh = () => { setRefreshing(true); load(); };

  // ── Order ────────────────────────────────────────────────────────────────────
  const submitOrder = async () => {
    const qty = parseFloat(oQty), price = parseFloat(oPrice);
    if (!oTicker) return Alert.alert('Error', 'Select a ticker.');
    if (!qty || qty <= 0) return Alert.alert('Error', 'Enter a valid quantity.');
    if (!price || price <= 0) return Alert.alert('Error', 'Enter a valid price.');
    try {
      await postOrder({ type: side.toLowerCase(), ticker: oTicker, qty, price });
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      setOrderModal(false);
      load(true);
    } catch (e) { Alert.alert('Order failed', e.message); }
  };

  // ── Stop loss ────────────────────────────────────────────────────────────────
  const submitStop = async () => {
    const qty = parseFloat(sQty), price = parseFloat(sPrice);
    if (!sTicker) return Alert.alert('Error', 'Select a ticker.');
    if (!qty || qty <= 0) return Alert.alert('Error', 'Enter a valid quantity.');
    if (!price || price <= 0) return Alert.alert('Error', 'Enter a valid stop price.');
    try {
      await postOrder({ type: 'stop_loss', ticker: sTicker, qty, price });
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      setStopModal(false);
      load(true);
    } catch (e) { Alert.alert('Failed', e.message); }
  };

  const onExecuteStop = (s) => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    Alert.alert(
      'Execute Stop Loss',
      `Sell ${fmt(s.qty, 4)} ${s.ticker} at stop $${fmt(s.stop_price)}?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Execute', style: 'destructive', onPress: async () => {
            try { await apiExecuteStop(s.id); Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success); load(true); }
            catch (e) { Alert.alert('Error', e.message); }
          }
        },
      ]
    );
  };

  const onDeleteStop = (s) => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    Alert.alert('Remove Stop Loss', `Delete stop on ${s.ticker} @ $${fmt(s.stop_price)}?`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete', style: 'destructive', onPress: async () => {
          try { await apiDeleteStop(s.id); load(true); }
          catch (e) { Alert.alert('Error', e.message); }
        }
      },
    ]);
  };

  // ── Derived ──────────────────────────────────────────────────────────────────
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

  // ── Render ───────────────────────────────────────────────────────────────────
  return (
    <View style={styles.root}>
      <StatusBar style="light" />

      {/* ── Header ── */}
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
        <ScalePress
          onPress={tab === 'portfolio' ? openOrder : openStop}
          style={[styles.headerBtn, tab !== 'portfolio' && { backgroundColor: RED }]}
        >
          <Text style={styles.headerBtnText}>{tab === 'portfolio' ? '+ Order' : '+ Stop Loss'}</Text>
        </ScalePress>
      </View>

      {/* ── Tab bar ── */}
      <View style={styles.tabBar}>
        {[
          { key: 'portfolio', label: '📊  Portfolio' },
          { key: 'stops', label: `🛡  Stops${stops.length ? `  ${stops.length}` : ''}` },
        ].map(({ key, label }) => (
          <TouchableOpacity
            key={key}
            style={[styles.tabItem, tab === key && styles.tabActive]}
            onPress={() => { Haptics.selectionAsync(); setTab(key); }}
          >
            <Text style={[styles.tabText, tab === key && styles.tabTextActive]}>{label}</Text>
          </TouchableOpacity>
        ))}
      </View>

      {/* Offline banner */}
      {connected === false && (
        <View style={styles.offlineBanner}>
          <Text style={styles.offlineText}>⚠  Cannot reach backend · {API_BASE}</Text>
        </View>
      )}

      {/* Initial loading */}
      {loading && holdings.length === 0 && (
        <View style={styles.centered}>
          <ActivityIndicator color={ACCENT} size="large" />
          <Text style={[styles.cardSub, { marginTop: 14 }]}>Connecting to backend…</Text>
        </View>
      )}

      {/* ── Portfolio tab ── */}
      {tab === 'portfolio' && !loading && (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={ACCENT} />}
        >
          <PortfolioHero holdings={positions} />

          <Text style={styles.sectionLabel}>POSITIONS</Text>
          {positions.length === 0 && (
            <Text style={[styles.cardSub, { textAlign: 'center', marginTop: 20 }]}>No positions yet.</Text>
          )}
          {positions.map((h) => <HoldingCard key={h.ticker} h={h} stops={stops} />)}

          {watchlist.length > 0 && (
            <>
              <Text style={[styles.sectionLabel, { marginTop: 28 }]}>WATCHLIST</Text>
              <View style={styles.watchRow}>
                {watchlist.map((h) => {
                  const dc = DECISION_COLOR[h.latest_prediction?.decision];
                  return (
                    <View key={h.ticker} style={[styles.watchChip, dc && { borderColor: dc + '55' }]}>
                      <Text style={styles.watchText}>{h.ticker}</Text>
                      {h.latest_prediction && (
                        <Text style={[styles.watchDecision, { color: dc || SUBTEXT }]}>
                          {h.latest_prediction.decision}
                        </Text>
                      )}
                    </View>
                  );
                })}
              </View>
            </>
          )}
        </ScrollView>
      )}

      {/* ── Stop Loss tab ── */}
      {tab === 'stops' && !loading && (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={ACCENT} />}
        >
          {stops.length === 0 ? (
            <View style={styles.empty}>
              <Text style={styles.emptyIcon}>🛡</Text>
              <Text style={styles.emptyText}>No stop-loss orders</Text>
              <Text style={styles.emptyHint}>Tap "+ Stop Loss" to protect a position.</Text>
            </View>
          ) : stops.map((s) => (
            <StopCard key={s.id} s={s} onExecute={onExecuteStop} onDelete={onDeleteStop} />
          ))}
        </ScrollView>
      )}

      {/* ══ ORDER MODAL ══ */}
      <Modal visible={orderModal} animationType="slide" transparent>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.overlay}>
          <View style={styles.sheet}>
            <View style={styles.sheetHandle} />
            <Text style={styles.sheetTitle}>New Order</Text>

            <View style={styles.toggle}>
              {['BUY', 'SELL'].map((s) => (
                <TouchableOpacity
                  key={s}
                  style={[styles.toggleBtn, side === s && (s === 'BUY' ? styles.toggleBuy : styles.toggleSell)]}
                  onPress={() => { Haptics.selectionAsync(); setSide(s); }}
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
            <TextInput style={styles.input} placeholder="e.g. 2.5" placeholderTextColor="#555"
              keyboardType="decimal-pad" value={oQty} onChangeText={setOQty} />
            <FieldLabel>Price per share ($)</FieldLabel>
            <TextInput style={styles.input} placeholder="e.g. 185.00" placeholderTextColor="#555"
              keyboardType="decimal-pad" value={oPrice} onChangeText={setOPrice} />

            <View style={styles.actions}>
              <TouchableOpacity style={styles.cancelBtn} onPress={() => setOrderModal(false)}>
                <Text style={styles.cancelText}>Cancel</Text>
              </TouchableOpacity>
              <ScalePress
                style={[styles.submitBtn, side === 'SELL' && { backgroundColor: RED }]}
                onPress={submitOrder}
              >
                <Text style={styles.submitText}>{side}</Text>
              </ScalePress>
            </View>
          </View>
        </KeyboardAvoidingView>
      </Modal>

      {/* ══ STOP LOSS MODAL ══ */}
      <Modal visible={stopModal} animationType="slide" transparent>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.overlay}>
          <View style={styles.sheet}>
            <View style={styles.sheetHandle} />
            <Text style={styles.sheetTitle}>New Stop-Loss</Text>
            <Text style={styles.sheetHint}>Set the price at which shares auto-sell.</Text>

            <TickerPicker
              label="Ticker (held positions only)"
              tickers={heldTickers.filter((t) => t.toLowerCase().includes(sSearch.toLowerCase()))}
              value={sTicker} search={sSearch} onSearch={setSSearch}
              open={sDropdown} setOpen={setSDrop} onSelect={setSTicker}
            />
            <FieldLabel>Qty to sell</FieldLabel>
            <TextInput style={styles.input} placeholder="e.g. 2.5" placeholderTextColor="#555"
              keyboardType="decimal-pad" value={sQty} onChangeText={setSQty} />
            <FieldLabel>Stop price ($)</FieldLabel>
            <TextInput style={styles.input} placeholder="e.g. 170.00" placeholderTextColor="#555"
              keyboardType="decimal-pad" value={sPrice} onChangeText={setSPrice} />

            <View style={styles.actions}>
              <TouchableOpacity style={styles.cancelBtn} onPress={() => setStopModal(false)}>
                <Text style={styles.cancelText}>Cancel</Text>
              </TouchableOpacity>
              <ScalePress style={[styles.submitBtn, { backgroundColor: RED }]} onPress={submitStop}>
                <Text style={styles.submitText}>Set Stop Loss</Text>
              </ScalePress>
            </View>
          </View>
        </KeyboardAvoidingView>
      </Modal>
    </View>
  );
}

// ─── Portfolio hero card ──────────────────────────────────────────────────────
function PortfolioHero({ holdings }) {
  let costBasis = 0, marketValue = 0, priceCount = 0;
  for (const h of holdings) {
    costBasis += h.qty * h.avg_cost;
    if (h.latest_prediction?.price) { marketValue += h.qty * h.latest_prediction.price; priceCount++; }
  }
  const pl = marketValue - costBasis;
  const plPct = costBasis > 0 ? (pl / costBasis) * 100 : 0;
  const plColor = pl >= 0 ? GREEN : RED;
  const sign = pl >= 0 ? '+' : '';
  const hasData = priceCount > 0;

  return (
    <View style={styles.hero}>
      <Text style={styles.heroLabel}>PORTFOLIO VALUE</Text>
      <Text style={styles.heroValue}>{hasData ? `$${fmt(marketValue)}` : '—'}</Text>
      {hasData && (
        <View style={styles.heroPLRow}>
          <Text style={[styles.heroPL, { color: plColor }]}>
            {sign}${fmt(Math.abs(pl))}
          </Text>
          <View style={[styles.heroPLBadge, { backgroundColor: plColor + '22' }]}>
            <Text style={[styles.heroPLPct, { color: plColor }]}>
              {sign}{fmt(Math.abs(plPct))}%
            </Text>
          </View>
        </View>
      )}
      <View style={styles.heroRow}>
        <View style={styles.heroStat}>
          <Text style={styles.heroStatLabel}>Cost Basis</Text>
          <Text style={styles.heroStatVal}>${fmt(costBasis)}</Text>
        </View>
        <View style={styles.heroStatDivider} />
        <View style={styles.heroStat}>
          <Text style={styles.heroStatLabel}>Positions</Text>
          <Text style={styles.heroStatVal}>{holdings.length}</Text>
        </View>
        <View style={styles.heroStatDivider} />
        <View style={styles.heroStat}>
          <Text style={styles.heroStatLabel}>Priced</Text>
          <Text style={styles.heroStatVal}>{priceCount}/{holdings.length}</Text>
        </View>
      </View>
    </View>
  );
}

// ─── Holding card (collapsible) ───────────────────────────────────────────────
function HoldingCard({ h, stops }) {
  const [open, setOpen] = useState(false);
  const [reasonOpen, setReason] = useState(false);
  const p = h.latest_prediction;
  const myStops = stops.filter((s) => s.ticker === h.ticker);
  const dc = DECISION_COLOR[p?.decision] || BORDER;

  const lastPrice = p?.price ?? null;
  const pl = lastPrice != null ? (lastPrice - h.avg_cost) * h.qty : null;
  const plPct = lastPrice != null && h.avg_cost > 0 ? ((lastPrice - h.avg_cost) / h.avg_cost) * 100 : null;
  const plColor = pl != null ? (pl >= 0 ? GREEN : RED) : SUBTEXT;
  const plSign = pl != null && pl >= 0 ? '+' : '';

  const trend = p?.trend60d ?? null;
  const trendColor = trend != null ? (trend >= 0 ? GREEN : RED) : SUBTEXT;
  const trendLabel = trend != null ? `${trend >= 0 ? '+' : ''}${fmt(trend, 1)}%` : null;

  const toggle = () => {
    LayoutAnimation.configureNext(LayoutAnimation.Presets.easeInEaseOut);
    Haptics.selectionAsync();
    setOpen((v) => !v);
  };

  return (
    <View style={[styles.card, { borderLeftWidth: 3, borderLeftColor: dc }]}>
      {/* ── Always-visible header row ── */}
      <Pressable onPress={toggle}>
        <View style={styles.cardHeader}>
          {/* Left: ticker + shares */}
          <View>
            <Text style={styles.cardTicker}>{h.ticker}</Text>
            <Text style={styles.cardQtySub}>{fmt(h.qty, 4)} shares · avg ${fmt(h.avg_cost)}</Text>
          </View>

          {/* Right: P&L + decision badge + chevron */}
          <View style={styles.cardHeaderRight}>
            {pl != null && (
              <Text style={[styles.plText, { color: plColor }]}>
                {plSign}{fmt(Math.abs(plPct), 1)}%
              </Text>
            )}
            {p && (
              <View style={[styles.decisionPill, { backgroundColor: dc + '33', borderColor: dc }]}>
                <Text style={[styles.decisionPillText, { color: dc }]}>{p.decision}</Text>
              </View>
            )}
            <Text style={styles.chevron}>{open ? '▲' : '▼'}</Text>
          </View>
        </View>
      </Pressable>

      {/* ── Expanded content ── */}
      {open && (
        <View style={styles.cardBody}>
          {/* P&L detail row */}
          {pl != null && (
            <View style={[styles.plRow, { backgroundColor: plColor + '11' }]}>
              <Text style={[styles.plRowMain, { color: plColor }]}>
                {plSign}${fmt(Math.abs(pl))}
              </Text>
              <Text style={[styles.plRowSub, { color: plColor }]}>
                {plSign}{fmt(Math.abs(plPct), 2)}% unrealized
              </Text>
              {trendLabel && (
                <View style={[styles.trendBadge, { borderColor: trendColor }]}>
                  <Text style={[styles.trendText, { color: trendColor }]}>{trendLabel} 60d</Text>
                </View>
              )}
            </View>
          )}

          {/* Prediction */}
          {p ? (
            <View style={styles.predSection}>
              <View style={styles.predTopRow}>
                <Text style={[styles.confidenceText, { color: CONFIDENCE_COLOR[p.confidence] || SUBTEXT }]}>
                  {p.confidence} confidence
                </Text>
                <Text style={styles.cardSub}>${fmt(p.price)}</Text>
              </View>

              {p.reasoning ? (
                <>
                  <Text style={styles.reasoningText} numberOfLines={reasonOpen ? undefined : 3}>
                    {p.reasoning}
                  </Text>
                  <TouchableOpacity onPress={() => setReason((v) => !v)}>
                    <Text style={styles.expandBtn}>{reasonOpen ? 'Show less ▲' : 'Read more ▼'}</Text>
                  </TouchableOpacity>
                </>
              ) : null}

              <View style={styles.targetsRow}>
                {p.priceTarget ? (
                  <View style={styles.targetChip}>
                    <Text style={styles.targetChipLabel}>TARGET</Text>
                    <Text style={styles.targetChipVal}>{p.priceTarget}</Text>
                  </View>
                ) : null}
                {p.stopLoss ? (
                  <View style={[styles.targetChip, { borderColor: RED + '55', backgroundColor: RED + '11' }]}>
                    <Text style={[styles.targetChipLabel, { color: RED }]}>AI STOP</Text>
                    <Text style={[styles.targetChipVal, { color: RED }]}>{p.stopLoss}</Text>
                  </View>
                ) : null}
              </View>

              <Text style={styles.predDate}>
                Predicted {new Date(p.date).toLocaleString()}
              </Text>
            </View>
          ) : (
            <Text style={[styles.cardSub, { marginTop: 10 }]}>No prediction yet.</Text>
          )}

          {/* Active stop-loss badges */}
          {myStops.length > 0 && (
            <View style={styles.stopBadgeRow}>
              {myStops.map((s) => (
                <View key={s.id} style={styles.stopBadge}>
                  <Text style={styles.stopBadgeText}>🛡 ${fmt(s.stop_price)} × {fmt(s.qty, 4)}</Text>
                </View>
              ))}
            </View>
          )}
        </View>
      )}
    </View>
  );
}

// ─── Stop loss card ───────────────────────────────────────────────────────────
function StopCard({ s, onExecute, onDelete }) {
  return (
    <View style={styles.stopCard}>
      <View style={styles.stopCardTop}>
        <View>
          <Text style={styles.cardTicker}>{s.ticker}</Text>
          <Text style={styles.cardSub}>Sell {fmt(s.qty, 4)} shares</Text>
          <Text style={styles.cardSub}>Set {new Date(s.created_at).toLocaleDateString()}</Text>
        </View>
        <View style={styles.stopPriceBlock}>
          <Text style={styles.stopPriceLabel}>STOP AT</Text>
          <Text style={styles.stopPriceValue}>${fmt(s.stop_price)}</Text>
        </View>
      </View>
      <View style={styles.stopActions}>
        <ScalePress style={styles.executeBtn} onPress={() => onExecute(s)}>
          <Text style={styles.executeBtnText}>⚡ Execute</Text>
        </ScalePress>
        <ScalePress style={styles.deleteBtn} onPress={() => onDelete(s)}>
          <Text style={styles.deleteBtnText}>✕ Delete</Text>
        </ScalePress>
      </View>
    </View>
  );
}

// ─── Helpers ──────────────────────────────────────────────────────────────────
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
          <TextInput style={styles.dropdownSearch} placeholder="Search…" placeholderTextColor="#555"
            value={search} onChangeText={onSearch} autoCapitalize="characters" />
          <FlatList
            data={tickers} keyExtractor={(t) => t} style={{ maxHeight: 160 }}
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

// ─── Styles ───────────────────────────────────────────────────────────────────
const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: DARK },

  // Header
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingTop: 60, paddingHorizontal: 16, paddingBottom: 14, borderBottomWidth: 1, borderBottomColor: BORDER },
  headerTitle: { color: TEXT, fontSize: 22, fontWeight: '800', letterSpacing: -0.5 },
  headerSub: { color: SUBTEXT, fontSize: 11, marginTop: 2 },
  headerBtn: { backgroundColor: ACCENT, paddingHorizontal: 16, paddingVertical: 9, borderRadius: 10 },
  headerBtnText: { color: '#000', fontWeight: '700', fontSize: 14 },

  // Tabs
  tabBar: { flexDirection: 'row', borderBottomWidth: 1, borderBottomColor: BORDER, backgroundColor: CARD },
  tabItem: { flex: 1, paddingVertical: 13, alignItems: 'center' },
  tabActive: { borderBottomWidth: 2, borderBottomColor: ACCENT },
  tabText: { color: SUBTEXT, fontSize: 13, fontWeight: '600' },
  tabTextActive: { color: ACCENT },

  // Banners
  offlineBanner: { backgroundColor: RED + '22', paddingVertical: 8, paddingHorizontal: 16, borderBottomWidth: 1, borderBottomColor: RED + '44' },
  offlineText: { color: RED, fontSize: 12, textAlign: 'center', fontWeight: '600' },

  centered: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  scroll: { padding: 16, paddingBottom: 48 },
  sectionLabel: { color: SUBTEXT, fontSize: 11, fontWeight: '700', letterSpacing: 1.2, marginBottom: 10 },

  // Hero
  hero: { backgroundColor: CARD, borderRadius: 16, padding: 20, marginBottom: 20, borderWidth: 1, borderColor: BORDER },
  heroLabel: { color: SUBTEXT, fontSize: 10, fontWeight: '700', letterSpacing: 1.5, marginBottom: 6 },
  heroValue: { color: TEXT, fontSize: 34, fontWeight: '800', letterSpacing: -1 },
  heroPLRow: { flexDirection: 'row', alignItems: 'center', gap: 8, marginTop: 4, marginBottom: 16 },
  heroPL: { fontSize: 18, fontWeight: '700' },
  heroPLBadge: { borderRadius: 6, paddingHorizontal: 8, paddingVertical: 3 },
  heroPLPct: { fontSize: 14, fontWeight: '700' },
  heroRow: { flexDirection: 'row', borderTopWidth: 1, borderTopColor: BORDER, paddingTop: 14, marginTop: 4 },
  heroStat: { flex: 1, alignItems: 'center' },
  heroStatDivider: { width: 1, backgroundColor: BORDER },
  heroStatLabel: { color: SUBTEXT, fontSize: 10, fontWeight: '600', letterSpacing: 0.5, marginBottom: 4 },
  heroStatVal: { color: TEXT, fontSize: 15, fontWeight: '700' },

  // Holding card
  card: { backgroundColor: CARD, borderRadius: 12, marginBottom: 10, borderWidth: 1, borderColor: BORDER, overflow: 'hidden' },
  cardHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: 14 },
  cardHeaderRight: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  cardTicker: { color: TEXT, fontSize: 17, fontWeight: '800' },
  cardQtySub: { color: SUBTEXT, fontSize: 12, marginTop: 2 },
  cardSub: { color: SUBTEXT, fontSize: 13 },
  chevron: { color: SUBTEXT, fontSize: 11, marginLeft: 2 },

  plText: { fontSize: 14, fontWeight: '700' },
  decisionPill: { borderWidth: 1, borderRadius: 6, paddingHorizontal: 8, paddingVertical: 3 },
  decisionPillText: { fontSize: 11, fontWeight: '800', letterSpacing: 0.5 },

  cardBody: { borderTopWidth: 1, borderTopColor: BORDER, paddingHorizontal: 14, paddingBottom: 14 },

  plRow: { flexDirection: 'row', alignItems: 'center', gap: 10, borderRadius: 8, padding: 10, marginTop: 12, marginBottom: 4 },
  plRowMain: { fontSize: 16, fontWeight: '800' },
  plRowSub: { fontSize: 12, fontWeight: '600', flex: 1 },
  trendBadge: { borderWidth: 1, borderRadius: 5, paddingHorizontal: 6, paddingVertical: 2 },
  trendText: { fontSize: 11, fontWeight: '700' },

  predSection: { marginTop: 12 },
  predTopRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 },
  confidenceText: { fontSize: 12, fontWeight: '700' },
  reasoningText: { color: SUBTEXT, fontSize: 13, lineHeight: 19, marginBottom: 4 },
  expandBtn: { color: ACCENT, fontSize: 12, fontWeight: '600', marginBottom: 10 },

  targetsRow: { flexDirection: 'row', gap: 8, marginTop: 4, marginBottom: 8 },
  targetChip: { borderWidth: 1, borderColor: GREEN + '55', backgroundColor: GREEN + '11', borderRadius: 7, paddingHorizontal: 10, paddingVertical: 6, alignItems: 'center' },
  targetChipLabel: { color: GREEN, fontSize: 9, fontWeight: '800', letterSpacing: 1 },
  targetChipVal: { color: GREEN, fontSize: 13, fontWeight: '700', marginTop: 1 },

  predDate: { color: SUBTEXT + '70', fontSize: 11, marginTop: 4 },

  stopBadgeRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 6, marginTop: 10 },
  stopBadge: { backgroundColor: RED + '18', borderWidth: 1, borderColor: RED + '44', borderRadius: 6, paddingHorizontal: 8, paddingVertical: 4 },
  stopBadgeText: { color: RED, fontSize: 11, fontWeight: '700' },

  // Stop card
  stopCard: { backgroundColor: CARD, borderRadius: 12, marginBottom: 10, borderWidth: 1, borderLeftWidth: 3, borderColor: BORDER, borderLeftColor: RED, overflow: 'hidden' },
  stopCardTop: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', padding: 14 },
  stopPriceBlock: { alignItems: 'flex-end' },
  stopPriceLabel: { color: RED, fontSize: 9, fontWeight: '800', letterSpacing: 1 },
  stopPriceValue: { color: RED, fontSize: 22, fontWeight: '800', marginTop: 2 },
  stopActions: { flexDirection: 'row', gap: 10, paddingHorizontal: 14, paddingBottom: 14 },
  executeBtn: { flex: 1, backgroundColor: YELLOW + '22', borderWidth: 1, borderColor: YELLOW + '88', borderRadius: 9, padding: 10, alignItems: 'center' },
  executeBtnText: { color: YELLOW, fontWeight: '700', fontSize: 13 },
  deleteBtn: { flex: 1, backgroundColor: RED + '18', borderWidth: 1, borderColor: RED + '55', borderRadius: 9, padding: 10, alignItems: 'center' },
  deleteBtnText: { color: RED, fontWeight: '700', fontSize: 13 },

  // Watchlist
  watchRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  watchChip: { backgroundColor: CARD, borderRadius: 8, paddingHorizontal: 12, paddingVertical: 7, borderWidth: 1, borderColor: BORDER, alignItems: 'center' },
  watchText: { color: TEXT, fontSize: 13, fontWeight: '700' },
  watchDecision: { fontSize: 10, fontWeight: '800', marginTop: 2, letterSpacing: 0.5 },

  // Empty
  empty: { alignItems: 'center', marginTop: 80 },
  emptyIcon: { fontSize: 40, marginBottom: 12 },
  emptyText: { color: SUBTEXT, fontSize: 17, fontWeight: '700' },
  emptyHint: { color: SUBTEXT + '80', fontSize: 13, marginTop: 6, textAlign: 'center' },

  // Modal / Sheet
  overlay: { flex: 1, justifyContent: 'flex-end', backgroundColor: 'rgba(0,0,0,0.65)' },
  sheet: { backgroundColor: CARD2, borderTopLeftRadius: 24, borderTopRightRadius: 24, padding: 24, paddingBottom: 44, borderTopWidth: 1, borderColor: BORDER },
  sheetHandle: { width: 36, height: 4, backgroundColor: BORDER, borderRadius: 2, alignSelf: 'center', marginBottom: 20 },
  sheetTitle: { color: TEXT, fontSize: 20, fontWeight: '800', marginBottom: 4 },
  sheetHint: { color: SUBTEXT, fontSize: 13, marginBottom: 12 },

  toggle: { flexDirection: 'row', marginTop: 16, borderRadius: 10, overflow: 'hidden', borderWidth: 1, borderColor: BORDER },
  toggleBtn: { flex: 1, paddingVertical: 12, alignItems: 'center', backgroundColor: DARK },
  toggleBuy: { backgroundColor: GREEN },
  toggleSell: { backgroundColor: RED },
  toggleText: { color: SUBTEXT, fontWeight: '700', fontSize: 14 },
  toggleTextActive: { color: '#fff', fontWeight: '800' },

  inputLabel: { color: SUBTEXT, fontSize: 12, fontWeight: '600', marginBottom: 6, marginTop: 14 },
  input: { backgroundColor: DARK, borderRadius: 10, borderWidth: 1, borderColor: BORDER, paddingHorizontal: 14, paddingVertical: 13, color: TEXT, fontSize: 15 },
  inputText: { color: TEXT, fontSize: 15 },
  inputPlaceholder: { color: SUBTEXT, fontSize: 15 },

  dropdown: { backgroundColor: DARK, borderWidth: 1, borderColor: BORDER, borderRadius: 10, marginTop: 4, overflow: 'hidden' },
  dropdownSearch: { borderBottomWidth: 1, borderBottomColor: BORDER, paddingHorizontal: 14, paddingVertical: 10, color: TEXT, fontSize: 14 },
  dropdownItem: { paddingHorizontal: 14, paddingVertical: 12 },
  dropdownItemText: { color: TEXT, fontSize: 14 },

  actions: { flexDirection: 'row', gap: 12, marginTop: 28 },
  cancelBtn: { flex: 1, paddingVertical: 14, borderRadius: 12, alignItems: 'center', borderWidth: 1, borderColor: BORDER },
  cancelText: { color: SUBTEXT, fontWeight: '700' },
  submitBtn: { flex: 2, paddingVertical: 14, borderRadius: 12, alignItems: 'center', backgroundColor: GREEN },
  submitText: { color: '#fff', fontWeight: '800', fontSize: 15 },
});
