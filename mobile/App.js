import AsyncStorage from '@react-native-async-storage/async-storage';
import { StatusBar } from 'expo-status-bar';
import { useEffect, useState } from 'react';
import {
  Alert,
  FlatList,
  KeyboardAvoidingView,
  Modal,
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';

const PORTFOLIO_KEY = '@jobot_portfolio';
const STOPS_KEY = '@jobot_stops';

const INITIAL_HOLDINGS = [
  { ticker: 'GLD', qty: 0, avgCost: 0 },
  { ticker: 'AMZN', qty: 8.0819, avgCost: 222.16 },
  { ticker: 'AMD', qty: 8.02, avgCost: 221.36 },
  { ticker: 'GOOG', qty: 4.1518, avgCost: 331.98 },
  { ticker: 'MSFT', qty: 2.66, avgCost: 412.24 },
  { ticker: 'BABA', qty: 4.6199, avgCost: 129.55 },
  { ticker: 'UBER', qty: 6.1534, avgCost: 81.05 },
  { ticker: 'ORCL', qty: 2.9628, avgCost: 162.86 },
  { ticker: 'NVDA', qty: 1.4292, avgCost: 180.06 },
  { ticker: 'META', qty: 0.3177, avgCost: 699.02 },
  { ticker: 'AAPL', qty: 0.525, avgCost: 256.44 },
  { ticker: 'TSM', qty: 0, avgCost: 0 },
  { ticker: 'TSLA', qty: 0, avgCost: 0 },
  { ticker: 'INTC', qty: 0, avgCost: 0 },
  { ticker: 'SNDK', qty: 0, avgCost: 0 },
];

const fmt = (n, decimals = 2) =>
  Number(n).toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });

// ─── colours ────────────────────────────────────────────────────────────────
const DARK = '#0d1117';
const CARD = '#161b22';
const BORDER = '#30363d';
const GREEN = '#3fb950';
const RED = '#f85149';
const YELLOW = '#d29922';
const ACCENT = '#58a6ff';
const TEXT = '#e6edf3';
const SUBTEXT = '#8b949e';

// ─── helpers ─────────────────────────────────────────────────────────────────
let stopIdSeq = 1;
const newStopId = () => `stop_${Date.now()}_${stopIdSeq++}`;

export default function App() {
  const [tab, setTab] = useState('portfolio'); // 'portfolio' | 'stops'

  // portfolio state
  const [holdings, setHoldings] = useState(INITIAL_HOLDINGS);
  const [orderModal, setOrderModal] = useState(false);
  const [side, setSide] = useState('BUY');
  const [oTicker, setOTicker] = useState('');
  const [oQty, setOQty] = useState('');
  const [oPrice, setOPrice] = useState('');
  const [oSearch, setOSearch] = useState('');
  const [oDropdown, setODropdown] = useState(false);

  // stop-loss state
  const [stops, setStops] = useState([]);
  const [stopModal, setStopModal] = useState(false);
  const [sTicker, setSTicker] = useState('');
  const [sQty, setSQty] = useState('');
  const [sPrice, setSPrice] = useState('');
  const [sSearch, setSSearch] = useState('');
  const [sDropdown, setSDdropdown] = useState(false);
  const [checkModal, setCheckModal] = useState(false);
  const [checkPrices, setCheckPrices] = useState({}); // ticker -> string

  // ── load from storage ──────────────────────────────────────────────────────
  useEffect(() => {
    AsyncStorage.multiGet([PORTFOLIO_KEY, STOPS_KEY]).then(([p, s]) => {
      if (p[1]) setHoldings(JSON.parse(p[1]));
      if (s[1]) setStops(JSON.parse(s[1]));
    });
  }, []);

  // ── save helpers ───────────────────────────────────────────────────────────
  const saveHoldings = (next) => {
    setHoldings(next);
    AsyncStorage.setItem(PORTFOLIO_KEY, JSON.stringify(next));
  };
  const saveStops = (next) => {
    setStops(next);
    AsyncStorage.setItem(STOPS_KEY, JSON.stringify(next));
  };

  // ── execute a stop (sell qty from holding) ─────────────────────────────────
  const executeStop = (stop, updatedHoldings) => {
    const base = updatedHoldings || holdings;
    const h = base.find((x) => x.ticker === stop.ticker);
    if (!h || h.qty < stop.qty) return base; // not enough shares, skip
    return base.map((x) =>
      x.ticker === stop.ticker
        ? { ...x, qty: x.qty - stop.qty, avgCost: x.qty - stop.qty === 0 ? 0 : x.avgCost }
        : x
    );
  };

  // ── order submit ───────────────────────────────────────────────────────────
  const submitOrder = () => {
    const qtyNum = parseFloat(oQty);
    const priceNum = parseFloat(oPrice);
    if (!oTicker) return Alert.alert('Error', 'Select a ticker.');
    if (!qtyNum || qtyNum <= 0) return Alert.alert('Error', 'Enter a valid quantity.');
    if (!priceNum || priceNum <= 0) return Alert.alert('Error', 'Enter a valid price.');

    const existing = holdings.find((h) => h.ticker === oTicker);

    if (side === 'BUY') {
      if (existing) {
        const newQty = existing.qty + qtyNum;
        const newAvg = (existing.qty * existing.avgCost + qtyNum * priceNum) / newQty;
        saveHoldings(holdings.map((h) =>
          h.ticker === oTicker ? { ...h, qty: newQty, avgCost: newAvg } : h
        ));
      } else {
        saveHoldings([...holdings, { ticker: oTicker, qty: qtyNum, avgCost: priceNum }]);
      }
    } else {
      if (!existing || existing.qty < qtyNum)
        return Alert.alert('Error', `Not enough shares. You hold ${existing?.qty ?? 0}.`);
      const newQty = existing.qty - qtyNum;
      saveHoldings(holdings.map((h) =>
        h.ticker === oTicker
          ? { ...h, qty: newQty, avgCost: newQty === 0 ? 0 : h.avgCost }
          : h
      ));
    }
    setOrderModal(false);
  };

  // ── stop-loss submit ───────────────────────────────────────────────────────
  const submitStop = () => {
    const qtyNum = parseFloat(sQty);
    const priceNum = parseFloat(sPrice);
    if (!sTicker) return Alert.alert('Error', 'Select a ticker.');
    if (!qtyNum || qtyNum <= 0) return Alert.alert('Error', 'Enter a valid quantity.');
    if (!priceNum || priceNum <= 0) return Alert.alert('Error', 'Enter a valid stop price.');

    const h = holdings.find((x) => x.ticker === sTicker);
    if (!h || h.qty < qtyNum)
      return Alert.alert('Error', `You only hold ${h?.qty ?? 0} shares of ${sTicker}.`);

    saveStops([...stops, { id: newStopId(), ticker: sTicker, qty: qtyNum, stopPrice: priceNum }]);
    setStopModal(false);
  };

  // ── manual execute stop ────────────────────────────────────────────────────
  const manualExecuteStop = (stop) => {
    Alert.alert(
      'Execute Stop Loss',
      `Sell ${fmt(stop.qty, 4)} ${stop.ticker} @ $${fmt(stop.stopPrice)}?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Execute',
          style: 'destructive',
          onPress: () => {
            const nextHoldings = executeStop(stop);
            saveHoldings(nextHoldings);
            saveStops(stops.filter((s) => s.id !== stop.id));
          },
        },
      ]
    );
  };

  const deleteStop = (id) => {
    Alert.alert('Delete Stop Loss', 'Remove this stop-loss order?', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', style: 'destructive', onPress: () => saveStops(stops.filter((s) => s.id !== id)) },
    ]);
  };

  // ── check prices and auto-trigger stops ───────────────────────────────────
  const runPriceCheck = () => {
    const triggered = stops.filter((s) => {
      const p = parseFloat(checkPrices[s.ticker]);
      return !isNaN(p) && p <= s.stopPrice;
    });

    if (triggered.length === 0) {
      Alert.alert('No stops triggered', 'None of your stop prices were hit.');
      setCheckModal(false);
      return;
    }

    let nextHoldings = [...holdings];
    triggered.forEach((s) => {
      nextHoldings = executeStop(s, nextHoldings);
    });

    const names = triggered.map((s) => `${s.ticker} (stop $${fmt(s.stopPrice)})`).join('\n');
    Alert.alert(
      `${triggered.length} Stop(s) Triggered`,
      `The following orders were executed:\n\n${names}`,
      [{
        text: 'OK',
        onPress: () => {
          saveHoldings(nextHoldings);
          saveStops(stops.filter((s) => !triggered.find((t) => t.id === s.id)));
          setCheckModal(false);
          setCheckPrices({});
        },
      }]
    );
  };

  // ── unique tickers for stop modal (only held positions) ────────────────────
  const heldTickers = holdings.filter((h) => h.qty > 0).map((h) => h.ticker);
  const allTickers = holdings.map((h) => h.ticker);
  const positions = holdings.filter((h) => h.qty > 0);
  const watchlist = holdings.filter((h) => h.qty === 0);
  const uniqueStopTickers = [...new Set(stops.map((s) => s.ticker))];

  // ── open order modal ───────────────────────────────────────────────────────
  const openOrder = () => {
    setSide('BUY'); setOTicker(''); setOQty(''); setOPrice('');
    setOSearch(''); setODropdown(false); setOrderModal(true);
  };

  // ── open stop modal ────────────────────────────────────────────────────────
  const openStop = () => {
    setSTicker(''); setSQty(''); setSPrice('');
    setSSearch(''); setSDdropdown(false); setStopModal(true);
  };

  // ── open price check modal ─────────────────────────────────────────────────
  const openCheck = () => {
    const initial = {};
    uniqueStopTickers.forEach((t) => { initial[t] = ''; });
    setCheckPrices(initial);
    setCheckModal(true);
  };

  // ─────────────────────────────────────────────────────────────────────────
  //  RENDER
  // ─────────────────────────────────────────────────────────────────────────
  return (
    <View style={styles.root}>
      <StatusBar style="light" />

      {/* ── Header ── */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Jobot</Text>
        {tab === 'portfolio' ? (
          <TouchableOpacity style={styles.headerBtn} onPress={openOrder}>
            <Text style={styles.headerBtnText}>+ Order</Text>
          </TouchableOpacity>
        ) : (
          <View style={{ flexDirection: 'row', gap: 8 }}>
            {stops.length > 0 && (
              <TouchableOpacity style={[styles.headerBtn, { backgroundColor: YELLOW }]} onPress={openCheck}>
                <Text style={[styles.headerBtnText, { color: '#000' }]}>Check Prices</Text>
              </TouchableOpacity>
            )}
            <TouchableOpacity style={[styles.headerBtn, { backgroundColor: RED }]} onPress={openStop}>
              <Text style={styles.headerBtnText}>+ Stop Loss</Text>
            </TouchableOpacity>
          </View>
        )}
      </View>

      {/* ── Tab bar ── */}
      <View style={styles.tabBar}>
        {[{ key: 'portfolio', label: 'Portfolio' }, { key: 'stops', label: `Stop Loss${stops.length ? ` (${stops.length})` : ''}` }].map(({ key, label }) => (
          <TouchableOpacity key={key} style={[styles.tabItem, tab === key && styles.tabActive]} onPress={() => setTab(key)}>
            <Text style={[styles.tabText, tab === key && styles.tabTextActive]}>{label}</Text>
          </TouchableOpacity>
        ))}
      </View>

      {/* ── Portfolio tab ── */}
      {tab === 'portfolio' && (
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={styles.sectionLabel}>POSITIONS</Text>
          {positions.map((h) => (
            <View key={h.ticker} style={styles.card}>
              <View style={styles.cardRow}>
                <Text style={styles.cardTicker}>{h.ticker}</Text>
                <Text style={styles.cardQty}>{fmt(h.qty, 4)} shares</Text>
              </View>
              <View style={styles.cardRow}>
                <Text style={styles.cardSub}>Avg cost</Text>
                <Text style={styles.cardSub}>${fmt(h.avgCost)}</Text>
              </View>
              <View style={styles.cardRow}>
                <Text style={styles.cardSub}>Cost basis</Text>
                <Text style={styles.cardSub}>${fmt(h.qty * h.avgCost)}</Text>
              </View>
              {stops.filter((s) => s.ticker === h.ticker).length > 0 && (
                <View style={styles.stopBadgeRow}>
                  {stops.filter((s) => s.ticker === h.ticker).map((s) => (
                    <View key={s.id} style={styles.stopBadge}>
                      <Text style={styles.stopBadgeText}>SL ${fmt(s.stopPrice)} × {fmt(s.qty, 4)}</Text>
                    </View>
                  ))}
                </View>
              )}
            </View>
          ))}

          <Text style={[styles.sectionLabel, { marginTop: 24 }]}>WATCHLIST</Text>
          <View style={styles.watchRow}>
            {watchlist.map((h) => (
              <View key={h.ticker} style={styles.watchChip}>
                <Text style={styles.watchText}>{h.ticker}</Text>
              </View>
            ))}
          </View>
        </ScrollView>
      )}

      {/* ── Stop Loss tab ── */}
      {tab === 'stops' && (
        <ScrollView contentContainerStyle={styles.scroll}>
          {stops.length === 0 ? (
            <View style={styles.empty}>
              <Text style={styles.emptyText}>No stop-loss orders.</Text>
              <Text style={styles.emptyHint}>Tap "+ Stop Loss" to add one.</Text>
            </View>
          ) : (
            stops.map((s) => (
              <View key={s.id} style={styles.stopCard}>
                <View style={styles.cardRow}>
                  <Text style={styles.cardTicker}>{s.ticker}</Text>
                  <View style={styles.stopPriceBadge}>
                    <Text style={styles.stopPriceText}>Stop @ ${fmt(s.stopPrice)}</Text>
                  </View>
                </View>
                <View style={styles.cardRow}>
                  <Text style={styles.cardSub}>Qty to sell</Text>
                  <Text style={styles.cardSub}>{fmt(s.qty, 4)} shares</Text>
                </View>
                <View style={[styles.cardRow, { marginTop: 10, gap: 8 }]}>
                  <TouchableOpacity style={styles.executeBtn} onPress={() => manualExecuteStop(s)}>
                    <Text style={styles.executeBtnText}>Execute</Text>
                  </TouchableOpacity>
                  <TouchableOpacity style={styles.deleteBtn} onPress={() => deleteStop(s.id)}>
                    <Text style={styles.deleteBtnText}>Delete</Text>
                  </TouchableOpacity>
                </View>
              </View>
            ))
          )}
        </ScrollView>
      )}

      {/* ════════════ ORDER MODAL ════════════ */}
      <Modal visible={orderModal} animationType="slide" transparent>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.overlay}>
          <View style={styles.sheet}>
            <Text style={styles.sheetTitle}>New Order</Text>
            <View style={styles.toggle}>
              {['BUY', 'SELL'].map((s) => (
                <TouchableOpacity key={s} style={[styles.toggleBtn, side === s && (s === 'BUY' ? styles.toggleBuy : styles.toggleSell)]} onPress={() => setSide(s)}>
                  <Text style={[styles.toggleText, side === s && styles.toggleTextActive]}>{s}</Text>
                </TouchableOpacity>
              ))}
            </View>
            <TickerPicker label="Ticker" tickers={allTickers.filter((t) => t.toLowerCase().includes(oSearch.toLowerCase()))} value={oTicker} search={oSearch} onSearch={setOSearch} open={oDropdown} setOpen={setODropdown} onSelect={setOTicker} />
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

      {/* ════════════ STOP LOSS MODAL ════════════ */}
      <Modal visible={stopModal} animationType="slide" transparent>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.overlay}>
          <View style={styles.sheet}>
            <Text style={styles.sheetTitle}>New Stop-Loss Order</Text>
            <Text style={styles.sheetHint}>When the price hits the stop, shares are sold.</Text>
            <TickerPicker label="Ticker" tickers={heldTickers.filter((t) => t.toLowerCase().includes(sSearch.toLowerCase()))} value={sTicker} search={sSearch} onSearch={setSSearch} open={sDropdown} setOpen={setSDdropdown} onSelect={setSTicker} />
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

      {/* ════════════ CHECK PRICES MODAL ════════════ */}
      <Modal visible={checkModal} animationType="slide" transparent>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : 'height'} style={styles.overlay}>
          <View style={styles.sheet}>
            <Text style={styles.sheetTitle}>Check Current Prices</Text>
            <Text style={styles.sheetHint}>Enter the current price for each ticker. Stops at or below the price will be triggered.</Text>
            {uniqueStopTickers.map((t) => (
              <View key={t}>
                <FieldLabel>{t}</FieldLabel>
                <TextInput style={styles.input} placeholder={`Current price of ${t}`} placeholderTextColor="#888" keyboardType="decimal-pad" value={checkPrices[t] || ''} onChangeText={(v) => setCheckPrices((prev) => ({ ...prev, [t]: v }))} />
              </View>
            ))}
            <View style={styles.actions}>
              <TouchableOpacity style={styles.cancelBtn} onPress={() => setCheckModal(false)}>
                <Text style={styles.cancelText}>Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity style={[styles.submitBtn, { backgroundColor: YELLOW }]} onPress={runPriceCheck}>
                <Text style={[styles.submitText, { color: '#000' }]}>Check</Text>
              </TouchableOpacity>
            </View>
          </View>
        </KeyboardAvoidingView>
      </Modal>
    </View>
  );
}

// ─── small helper components ─────────────────────────────────────────────────
function FieldLabel({ children }) {
  return <Text style={styles.inputLabel}>{children}</Text>;
}

function TickerPicker({ label, tickers, value, search, onSearch, open, setOpen, onSelect }) {
  return (
    <>
      <FieldLabel>{label}</FieldLabel>
      <TouchableOpacity style={styles.input} onPress={() => setOpen(!open)}>
        <Text style={value ? styles.inputText : styles.inputPlaceholder}>{value || 'Select ticker…'}</Text>
      </TouchableOpacity>
      {open && (
        <View style={styles.dropdown}>
          <TextInput style={styles.dropdownSearch} placeholder="Search…" placeholderTextColor="#888" value={search} onChangeText={onSearch} autoCapitalize="characters" />
          <FlatList data={tickers} keyExtractor={(t) => t} style={{ maxHeight: 150 }} renderItem={({ item }) => (
            <TouchableOpacity style={styles.dropdownItem} onPress={() => { onSelect(item); setOpen(false); onSearch(''); }}>
              <Text style={styles.dropdownItemText}>{item}</Text>
            </TouchableOpacity>
          )} />
        </View>
      )}
    </>
  );
}

// ─── styles ──────────────────────────────────────────────────────────────────
const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: DARK },

  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingTop: 60, paddingHorizontal: 16, paddingBottom: 12, borderBottomWidth: 1, borderBottomColor: BORDER },
  headerTitle: { color: TEXT, fontSize: 20, fontWeight: '700' },
  headerBtn: { backgroundColor: ACCENT, paddingHorizontal: 14, paddingVertical: 7, borderRadius: 8 },
  headerBtnText: { color: '#000', fontWeight: '700', fontSize: 14 },

  tabBar: { flexDirection: 'row', borderBottomWidth: 1, borderBottomColor: BORDER },
  tabItem: { flex: 1, paddingVertical: 12, alignItems: 'center' },
  tabActive: { borderBottomWidth: 2, borderBottomColor: ACCENT },
  tabText: { color: SUBTEXT, fontSize: 14, fontWeight: '600' },
  tabTextActive: { color: ACCENT },

  scroll: { padding: 16, paddingBottom: 40 },
  sectionLabel: { color: SUBTEXT, fontSize: 11, fontWeight: '600', letterSpacing: 1, marginBottom: 8 },

  card: { backgroundColor: CARD, borderRadius: 10, padding: 14, marginBottom: 10, borderWidth: 1, borderColor: BORDER },
  stopCard: { backgroundColor: CARD, borderRadius: 10, padding: 14, marginBottom: 10, borderWidth: 1, borderColor: RED + '55' },
  cardRow: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 4 },
  cardTicker: { color: TEXT, fontSize: 16, fontWeight: '700' },
  cardQty: { color: ACCENT, fontSize: 14, fontWeight: '600' },
  cardSub: { color: SUBTEXT, fontSize: 13 },

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
  watchChip: { backgroundColor: CARD, borderRadius: 6, paddingHorizontal: 10, paddingVertical: 5, borderWidth: 1, borderColor: BORDER },
  watchText: { color: SUBTEXT, fontSize: 13, fontWeight: '600' },

  empty: { alignItems: 'center', marginTop: 80 },
  emptyText: { color: SUBTEXT, fontSize: 16, fontWeight: '600' },
  emptyHint: { color: SUBTEXT + '88', fontSize: 13, marginTop: 6 },

  overlay: { flex: 1, justifyContent: 'flex-end', backgroundColor: 'rgba(0,0,0,0.6)' },
  sheet: { backgroundColor: CARD, borderTopLeftRadius: 20, borderTopRightRadius: 20, padding: 24, paddingBottom: 40, borderTopWidth: 1, borderColor: BORDER },
  sheetTitle: { color: TEXT, fontSize: 18, fontWeight: '700', marginBottom: 6 },
  sheetHint: { color: SUBTEXT, fontSize: 13, marginBottom: 16 },

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
