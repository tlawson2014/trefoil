// Everything that touches the chain lives here: keys, addresses, building
// and signing transactions, and the handful of REST calls the wallet needs.
//
// The rule that matters most: the private key never leaves this file's
// in-memory wallet object. We sign here and send only the signed bytes.

import {
  DirectSecp256k1HdWallet,
  Registry,
  makeAuthInfoBytes,
  makeSignDoc,
  encodePubkey,
} from "@cosmjs/proto-signing";
import { encodeSecp256k1Pubkey } from "@cosmjs/amino";
import { toBase64, fromBase64, toHex } from "@cosmjs/encoding";
import { sha256 } from "@cosmjs/crypto";
import { TxRaw } from "cosmjs-types/cosmos/tx/v1beta1/tx";
import { MsgSend } from "cosmjs-types/cosmos/bank/v1beta1/tx";
import _m0 from "protobufjs/minimal.js";

export const CONFIG = {
  // The node's REST API. Ignite's `chain serve` exposes it on 1317.
  // Anything that isn't localhost must be https.
  rest: localStorage.getItem("trefoil.rest") || "http://localhost:1317",
  chainId: localStorage.getItem("trefoil.chainId") || "trefoil",
  prefix: "trefoil",
  denom: "utfl",
  symbol: "TFL",
  decimals: 6,
  gas: 250000,
};

// ---------------------------------------------------------------------------
// Trefoil's own messages. The chain generated these from proto files; the
// wallet needs matching encoders. Field numbers must match the .proto exactly.
// ---------------------------------------------------------------------------

function stringsMsg(typeUrl, fields) {
  // Generic encoder for messages made of string / repeated string / uint64
  // fields, which is all of x/recovery's messages.
  return {
    typeUrl,
    encode(m, w = _m0.Writer.create()) {
      for (const [name, no, kind] of fields) {
        const v = m[name];
        if (kind === "string" && v) w.uint32((no << 3) | 2).string(v);
        if (kind === "strings") for (const s of v || []) w.uint32((no << 3) | 2).string(s);
        if (kind === "uint64" && v) w.uint32((no << 3) | 0).uint64(Number(v));
        if (kind === "bool" && v) w.uint32((no << 3) | 0).bool(true);
      }
      return w;
    },
    decode(input, length) {
      const r = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
      const end = length === undefined ? r.len : r.pos + length;
      const m = {};
      for (const [name, , kind] of fields) m[name] = kind === "strings" ? [] : kind === "uint64" ? 0 : kind === "bool" ? false : "";
      while (r.pos < end) {
        const tag = r.uint32();
        const f = fields.find(([, no]) => no === tag >>> 3);
        if (!f) { r.skipType(tag & 7); continue; }
        const [name, , kind] = f;
        if (kind === "string") m[name] = r.string();
        else if (kind === "strings") m[name].push(r.string());
        else if (kind === "bool") m[name] = r.bool();
        else m[name] = Number(r.uint64());
      }
      return m;
    },
    fromPartial(o) { return { ...o }; },
  };
}

// Field numbers match proto/trefoil/undo/v1/tx.proto exactly.
export const MsgDelayedSend = {
  typeUrl: "/trefoil.undo.v1.MsgDelayedSend",
  encode(m, w = _m0.Writer.create()) {
    if (m.creator) w.uint32(10).string(m.creator);
    if (m.recipient) w.uint32(18).string(m.recipient);
    // amount is a cosmos.base.v1beta1.Coin: {denom=1, amount=2}
    const c = _m0.Writer.create();
    if (m.amount?.denom) c.uint32(10).string(m.amount.denom);
    if (m.amount?.amount) c.uint32(18).string(m.amount.amount);
    w.uint32(26).bytes(c.finish());
    if (m.window) w.uint32(32).uint64(Number(m.window));
    return w;
  },
  decode(input, length) {
    const r = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    const end = length === undefined ? r.len : r.pos + length;
    const m = { creator: "", recipient: "", amount: { denom: "", amount: "" }, window: 0 };
    while (r.pos < end) {
      const tag = r.uint32();
      switch (tag >>> 3) {
        case 1: m.creator = r.string(); break;
        case 2: m.recipient = r.string(); break;
        case 3: {
          const len = r.uint32(), stop = r.pos + len;
          while (r.pos < stop) {
            const t = r.uint32();
            if (t >>> 3 === 1) m.amount.denom = r.string();
            else if (t >>> 3 === 2) m.amount.amount = r.string();
            else r.skipType(t & 7);
          }
          break;
        }
        case 4: m.window = Number(r.uint64()); break;
        default: r.skipType(tag & 7);
      }
    }
    return m;
  },
  fromPartial(o) { return { ...o }; },
};

export const MsgCancelSend = stringsMsg("/trefoil.undo.v1.MsgCancelSend", [
  ["creator", 1, "string"], ["id", 2, "uint64"],
]);
export const MsgSetFinalOnly = stringsMsg("/trefoil.undo.v1.MsgSetFinalOnly", [
  ["creator", 1, "string"], ["finalonly", 2, "bool"],
]);

const registry = new Registry([
  ["/cosmos.bank.v1beta1.MsgSend", MsgSend],
  [MsgDelayedSend.typeUrl, MsgDelayedSend],
  [MsgCancelSend.typeUrl, MsgCancelSend],
  [MsgSetFinalOnly.typeUrl, MsgSetFinalOnly],
]);

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

export async function newWallet() {
  return DirectSecp256k1HdWallet.generate(12, { prefix: CONFIG.prefix });
}

export async function walletFromMnemonic(mnemonic) {
  // Accept the numbered form the copy button produces ("1. woman\n2. cabbage…")
  // as well as plain words.
  const words = mnemonic.toLowerCase().replace(/\d+\.?/g, " ").split(/[^a-z]+/).filter(Boolean).join(" ");
  return DirectSecp256k1HdWallet.fromMnemonic(words, { prefix: CONFIG.prefix });
}

export async function addressOf(wallet) {
  const [acc] = await wallet.getAccounts();
  return acc.address;
}

export function isAddress(s) {
  return typeof s === "string" && /^trefoil1[02-9ac-hj-np-z]{38}$/.test(s.trim());
}

// ---------------------------------------------------------------------------
// REST reads
// ---------------------------------------------------------------------------

async function get(path) {
  const res = await fetch(CONFIG.rest + path);
  if (res.status === 404) return null;
  const body = await res.json();
  if (!res.ok) throw new Error(body.message || `HTTP ${res.status}`);
  return body;
}

export async function getBalance(address) {
  const b = await get(`/cosmos/bank/v1beta1/balances/${address}`);
  const c = (b?.balances || []).find((x) => x.denom === CONFIG.denom);
  return c ? BigInt(c.amount) : 0n;
}

// Returns {accountNumber, sequence} or null if the chain has never seen it.
export async function getAccount(address) {
  try {
    const a = await get(`/cosmos/auth/v1beta1/accounts/${address}`);
    if (!a?.account) return null;
    const acc = a.account.base_account || a.account;
    return { accountNumber: Number(acc.account_number), sequence: Number(acc.sequence) };
  } catch (e) {
    if (/not found|NotFound|does not exist/i.test(e.message)) return null;
    throw e;
  }
}

// The public tally for an address: {index, sent, cancelled, finalonly}.
// An address that has never used undo has no record; we return zeros.
export async function getRecord(address) {
  try {
    const r = await get(`/trefoil/undo/v1/record/${address}`);
    const rec = r?.record;
    return rec ? { sent: Number(rec.sent || 0), cancelled: Number(rec.cancelled || 0), finalonly: !!rec.finalonly } : { sent: 0, cancelled: 0, finalonly: false };
  } catch (e) {
    if (/not found/i.test(e.message)) return { sent: 0, cancelled: 0, finalonly: false };
    throw e;
  }
}

// Every pending send that involves this address, split into what it is
// sending and what is coming to it. A full table walk — fine for a
// prototype; a real deployment would index by address.
export async function pendingFor(address) {
  const out = { outgoing: [], incoming: [] };
  let key = null;
  do {
    const q = key ? `?pagination.key=${encodeURIComponent(key)}` : "";
    const page = await get(`/trefoil/undo/v1/pending${q}`);
    for (const p of page?.pending || []) {
      const row = { id: Number(p.id), sender: p.sender, recipient: p.recipient, amount: BigInt(p.amount?.amount || 0), executeat: Number(p.executeat) };
      if (p.sender === address) out.outgoing.push(row);
      if (p.recipient === address) out.incoming.push(row);
    }
    key = page?.pagination?.next_key || null;
  } while (key);
  out.outgoing.sort((a, b) => a.executeat - b.executeat);
  out.incoming.sort((a, b) => a.executeat - b.executeat);
  return out;
}

export async function getParams() {
  const p = await get(`/trefoil/undo/v1/params`);
  const x = p?.params || {};
  return { min: Number(x.minwindow || 120), def: Number(x.defaultwindow || 600), max: Number(x.maxwindow || 86400) };
}

// ---------------------------------------------------------------------------
// History, read from the chain itself
//
// A Cosmos chain stores blocks, not "everything address X ever did" — so there
// is normally an indexer service in between. CometBFT does keep a small index
// of transactions by tag, which is enough for a wallet: we ask the node for
// every transaction this address signed, then read the events our own module
// emitted inside them (delayed_send, send_cancelled, final_send). Those events
// carry the id, the recipient and the amount, so the list can be rebuilt on any
// device from the 12 words alone.
//
// It can fail — a pruned node may have dropped old blocks, and a node can be
// configured with the index off — so the wallet keeps a local copy as a
// fallback and never depends on this working.
export async function historyFromChain(address, limit = 40) {
  const q = encodeURIComponent(`message.sender='${address}'`);
  let body = null;
  try {
    body = await get(`/cosmos/tx/v1beta1/txs?query=${q}&order_by=ORDER_BY_DESC&pagination.limit=${limit}`);
  } catch {
    // Older SDK builds spell the parameter differently.
    body = await get(`/cosmos/tx/v1beta1/txs?events=${q}&order_by=ORDER_BY_DESC&pagination.limit=${limit}`);
  }
  const rows = body?.tx_responses || [];
  const out = [];
  const cancelled = new Set();

  for (const r of rows) {
    if (Number(r.code) !== 0) continue;               // refused transactions aren't history
    const at = r.timestamp ? Date.parse(r.timestamp) : Date.now();
    for (const ev of r.events || []) {
      const a = {};
      for (const kv of ev.attributes || []) a[kv.key] = kv.value;
      if (ev.type === "delayed_send") {
        out.push({ kind: "sent", id: Number(a.id), addr: a.recipient, amount: amountOf(a.amount), at, status: "pending" });
      } else if (ev.type === "final_send") {
        out.push({ kind: "sent", id: null, addr: a.recipient, amount: amountOf(a.amount), at, status: "final" });
      } else if (ev.type === "send_cancelled") {
        cancelled.add(Number(a.id));
      }
    }
  }
  for (const h of out) if (h.id != null && cancelled.has(h.id)) h.status = "cancelled";
  return out.sort((x, y) => y.at - x.at);
}

// Events carry amounts as a coin string, e.g. "100000000utfl".
function amountOf(s) {
  const m = String(s || "").match(/^(\d+)/);
  return m ? m[1] : "0";
}

export async function getTx(hash) {
  return get(`/cosmos/tx/v1beta1/txs/${hash}`);
}

export async function latestHeight() {
  const b = await get(`/cosmos/base/tendermint/v1beta1/blocks/latest`);
  return Number(b?.block?.header?.height || 0);
}

// ---------------------------------------------------------------------------
// Sign and send
// ---------------------------------------------------------------------------

// sendTx signs `msgs` from `wallet` and broadcasts. Returns the tx hash
// once the tx is in a block, or throws with a human-readable reason.
export async function sendTx(wallet, msgs, { onStatus } = {}) {
  const address = await addressOf(wallet);
  const [acc] = await wallet.getAccounts();

  const account = await getAccount(address);
  if (!account) throw new Error("This wallet has no coins yet. Ask someone to send you some first.");

  onStatus?.("Signing…");
  const bodyBytes = registry.encodeTxBody({ messages: msgs, memo: "" });
  const pubkey = encodePubkey(encodeSecp256k1Pubkey(acc.pubkey));
  const authInfoBytes = makeAuthInfoBytes([{ pubkey, sequence: account.sequence }], [], CONFIG.gas);
  const signDoc = makeSignDoc(bodyBytes, authInfoBytes, CONFIG.chainId, account.accountNumber);
  const { signature, signed } = await wallet.signDirect(address, signDoc);
  const raw = TxRaw.encode(TxRaw.fromPartial({
    bodyBytes: signed.bodyBytes,
    authInfoBytes: signed.authInfoBytes,
    signatures: [fromBase64(signature.signature)],
  })).finish();

  onStatus?.("Sending…");
  const res = await fetch(CONFIG.rest + "/cosmos/tx/v1beta1/txs", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ tx_bytes: toBase64(raw), mode: "BROADCAST_MODE_SYNC" }),
  });
  const body = await res.json();
  const r = body.tx_response;
  if (!r) throw new Error(body.message || "Node rejected the transaction");
  if (r.code !== 0) throw new Error(friendly(r.raw_log || r.log || `code ${r.code}`));

  onStatus?.("Waiting for the next block…");
  const hash = r.txhash || toHex(sha256(raw)).toUpperCase();
  for (let i = 0; i < 30; i++) {
    await new Promise((f) => setTimeout(f, 1000));
    const t = await getTx(hash).catch(() => null);
    if (t?.tx_response) {
      if (t.tx_response.code !== 0) throw new Error(friendly(t.tx_response.raw_log));
      return hash;
    }
  }
  throw new Error("The transaction was sent but hasn't appeared in a block yet. Check again in a moment.");
}

// Turn the chain's error text into something a person can act on.
function friendly(log) {
  const s = String(log);
  if (/insufficient funds/i.test(s)) return "Not enough TFL for that.";
  if (/only accepts final payments/i.test(s)) return "That address only accepts final payments — send with no undo window.";
  if (/shorter than the chain minimum/i.test(s)) return "The undo window is too short. Minimum is 2 minutes.";
  if (/longer than the chain maximum/i.test(s)) return "The undo window is too long. Maximum is 24 hours.";
  if (/no pending send/i.test(s)) return "Too late — that payment is already final.";
  if (/only the sender can undo/i.test(s)) return "Only the sender can undo a payment.";
  if (/too many undoable sends/i.test(s)) return "You have too many payments waiting. Let some go through first.";
  if (/cannot send to yourself/i.test(s)) return "You can't send to yourself.";
  if (/account sequence mismatch/i.test(s)) return "The chain is still processing your last action. Try again in a few seconds.";
  if (/signature verification failed/i.test(s)) return "The signature didn't verify. Is the wallet pointed at the right chain?";
  return s.length > 200 ? s.slice(0, 200) + "…" : s;
}

// ---------------------------------------------------------------------------
// Amounts
// ---------------------------------------------------------------------------

export function fmt(micro) {
  const n = BigInt(micro);
  const whole = n / 1_000_000n;
  const frac = (n % 1_000_000n).toString().padStart(6, "0").replace(/0+$/, "");
  return whole.toLocaleString("en-GB") + (frac ? "." + frac : "");
}

export function parseAmount(s) {
  const m = String(s).trim().match(/^(\d+)(?:\.(\d{1,6}))?$/);
  if (!m) return null;
  return BigInt(m[1]) * 1_000_000n + BigInt((m[2] || "").padEnd(6, "0"));
}

export function short(addr) {
  return addr ? addr.slice(0, 11) + "…" + addr.slice(-5) : "";
}

// "8:41" / "1h 03m" / "23h 59m" for a seconds-from-now value.
export function countdown(seconds) {
  const s = Math.max(0, Math.floor(seconds));
  if (s >= 3600) return `${Math.floor(s / 3600)}h ${String(Math.floor((s % 3600) / 60)).padStart(2, "0")}m`;
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}
