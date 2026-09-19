// The screens. Each `show(name)` swaps which <section> is visible; state
// lives in `S`. Wording follows docs/WALLET-FLOW.md — if you change words
// there, change them here.

import QRCode from "qrcode";
import {
  CONFIG, newWallet, walletFromMnemonic, addressOf, isAddress,
  getBalance, getRecord, pendingFor, getParams, latestHeight,
  sendTx, MsgDelayedSend, MsgCancelSend, MsgSetFinalOnly,
  fmt, parseAmount, short, countdown,
} from "./chain.js";

const STORE = "trefoil.wallet";
const $ = (id) => document.getElementById(id);
const S = { wallet: null, address: null, params: { min: 120, def: 600, max: 86400 }, pending: { outgoing: [], incoming: [] }, record: null };

// ---------------------------------------------------------------------------
// Storage (prototype: the recovery phrase is kept in browser storage as-is.
// v1.1 adds a password. Never store it anywhere else.)
// ---------------------------------------------------------------------------

function load() {
  try { return JSON.parse(localStorage.getItem(STORE) || "null"); } catch { return null; }
}
function save(data) {
  try { localStorage.setItem(STORE, JSON.stringify(data)); } catch {}
}
function persist() { save({ mnemonic: S.wallet?.mnemonic }); }

// ---------------------------------------------------------------------------
// Screens
// ---------------------------------------------------------------------------

function show(name) {
  document.querySelectorAll("section[data-screen]").forEach((s) => (s.hidden = s.dataset.screen !== name));
  window.scrollTo(0, 0);
}
function toast(msg, isError = false) {
  const t = $("toast");
  t.textContent = msg;
  t.className = isError ? "toast error" : "toast";
  t.hidden = false;
  clearTimeout(t._t);
  t._t = setTimeout(() => (t.hidden = true), isError ? 6000 : 3000);
}
function busy(btn, on, label) {
  btn.disabled = on;
  if (label) btn.textContent = label;
}
function copy(text) {
  navigator.clipboard?.writeText(text).then(() => toast("Copied.")).catch(() => toast(text));
}

// --- 1. Welcome ---------------------------------------------------------------

$("btn-create").onclick = async () => {
  S.wallet = await newWallet();
  S.address = await addressOf(S.wallet);
  renderPhrase(S.wallet.mnemonic, "phrase-words");
  show("phrase");
};
$("btn-restore").onclick = () => show("restore");

// --- 2. Recovery phrase --------------------------------------------------------

function renderPhrase(mnemonic, el) {
  $(el).innerHTML = mnemonic.split(" ").map((w, i) => `<li><span>${i + 1}</span>${w}</li>`).join("");
}
// Copy with numbers, one per line, so what's pasted reads like the screen.
function numbered(mnemonic) {
  return mnemonic.split(" ").map((w, i) => `${i + 1}. ${w}`).join("\n");
}
$("btn-phrase-copy").onclick = () => copy(numbered(S.wallet.mnemonic));
$("btn-phrase-written").onclick = () => {
  setupConfirm(S.wallet.mnemonic);
  show("confirm");
};

// --- 3. Confirm phrase ---------------------------------------------------------

let confirmTarget = [];
function setupConfirm(mnemonic) {
  const words = mnemonic.split(" ");
  const picks = [2, 6, 10]; // words 3, 7, 11
  confirmTarget = picks.map((i) => words[i]);
  $("confirm-ask").textContent = `Tap word ${picks[0] + 1}, then word ${picks[1] + 1}, then word ${picks[2] + 1}.`;
  const shuffled = [...words].sort(() => Math.random() - 0.5);
  $("confirm-grid").innerHTML = shuffled.map((w) => `<button type="button" class="chip">${w}</button>`).join("");
  $("confirm-progress").textContent = "";
  let got = [];
  $("confirm-grid").querySelectorAll("button").forEach((b) => {
    b.onclick = () => {
      got.push(b.textContent);
      b.disabled = true;
      $("confirm-progress").textContent = got.join(" · ");
      if (got.length === 3) {
        if (got.join() === confirmTarget.join()) {
          persist();
          toast("Wallet created.");
          enterHome();
        } else {
          toast("Not quite — check your notes and try again.", true);
          setupConfirm(mnemonic);
        }
      }
    };
  });
}

// --- Restore -----------------------------------------------------------------------

$("btn-restore-go").onclick = async (e) => {
  try {
    busy(e.target, true, "Restoring…");
    S.wallet = await walletFromMnemonic($("restore-phrase").value);
    S.address = await addressOf(S.wallet);
    persist();
    enterHome();
  } catch {
    toast("That doesn't look like a valid 12-word phrase.", true);
  } finally {
    busy(e.target, false, "Restore");
  }
};
$("btn-back-welcome").onclick = () => show("welcome");

// --- Home --------------------------------------------------------------------------

async function enterHome() {
  show("home");
  $("home-addr").textContent = short(S.address);
  $("home-addr-full").textContent = S.address;
  await refresh();
}

async function refresh() {
  try {
    const [bal, pending, record] = await Promise.all([
      getBalance(S.address), pendingFor(S.address), getRecord(S.address),
    ]);
    S.pending = pending;
    S.record = record;
    $("balance").textContent = fmt(bal);
    renderPending();
    $("finalonly-banner").hidden = !record.finalonly;
    $("net-status").textContent = "Connected to " + CONFIG.rest.replace(/^https?:\/\//, "");
    $("net-status").className = "muted";
  } catch (err) {
    $("net-status").textContent = "Can't reach the chain at " + CONFIG.rest + " — is it running?";
    $("net-status").className = "muted error";
  }
}
// Re-read the chain every 6 s (a block) and redraw the countdowns every second.
setInterval(() => { if (!$("home").hidden) refresh(); }, 6000);
setInterval(() => { if (!$("home").hidden) tick(); }, 1000);

// Chain time vs browser time: block timestamps are UTC seconds; the browser
// clock is close enough for a countdown.
const now = () => Math.floor(Date.now() / 1000);

function renderPending() {
  const { outgoing, incoming } = S.pending;
  $("outgoing-wrap").hidden = outgoing.length === 0;
  $("incoming-wrap").hidden = incoming.length === 0;

  $("outgoing").innerHTML = outgoing.map((p) => `
    <li data-id="${p.id}" data-at="${p.executeat}">
      <div><span class="amt">${fmt(p.amount)} TFL</span> to <code>${short(p.recipient)}</code><br>
        <span class="muted">arriving in <span class="cd"></span></span></div>
      <button class="secondary" data-undo="${p.id}">Undo</button>
    </li>`).join("");
  $("outgoing").querySelectorAll("[data-undo]").forEach((b) => (b.onclick = () => undo(Number(b.dataset.undo), b)));

  $("incoming").innerHTML = incoming.map((p) => `
    <li class="in" data-at="${p.executeat}">
      <div><span class="amt">${fmt(p.amount)} TFL</span> promised by <code>${short(p.sender)}</code><br>
        <span>yours in <span class="cd"></span></span></div>
    </li>`).join("");
  tick();
}

function tick() {
  const t = now();
  document.querySelectorAll(".pend li[data-at]").forEach((li) => {
    const left = Number(li.dataset.at) - t;
    const cd = li.querySelector(".cd");
    if (left <= 0) {
      cd.textContent = "next block";
      const b = li.querySelector("button");
      if (b) { b.disabled = true; b.textContent = "Final"; }
    } else {
      cd.textContent = countdown(left);
    }
  });
}

async function undo(id, btn) {
  const p = S.pending.outgoing.find((x) => x.id === id);
  if (!p) return;
  if (!confirm(`Take back ${fmt(p.amount)} TFL from ${short(p.recipient)}?`)) return;
  busy(btn, true, "Undoing…");
  try {
    await sendTx(S.wallet, [{ typeUrl: MsgCancelSend.typeUrl, value: { creator: S.address, id } }],
      { onStatus: (s) => (btn.textContent = s) });
    toast(`Took back ${fmt(p.amount)} TFL.`);
    await refresh();
  } catch (err) {
    toast(err.message, true);
    busy(btn, false, "Undo");
    refresh();
  }
}

// --- Send ------------------------------------------------------------------------------

let sendWindow = 600;
let recipientInfo = null;

$("window-seg").querySelectorAll("button").forEach((b) => {
  b.onclick = () => {
    sendWindow = Number(b.dataset.w);
    $("window-seg").querySelectorAll("button").forEach((x) => x.classList.toggle("on", x === b));
  };
});

$("btn-send").onclick = () => {
  $("send-to").value = ""; $("send-amount").value = "";
  $("recipient-note").hidden = true; recipientInfo = null;
  show("send");
};
$("btn-send-back").onclick = () => show("home");

// As soon as a valid address is typed, look up what the chain knows about it.
$("send-to").oninput = async () => {
  const to = $("send-to").value.trim();
  recipientInfo = null;
  $("recipient-note").hidden = true;
  if (!isAddress(to)) return;
  try {
    const r = await getRecord(to);
    if ($("send-to").value.trim() !== to) return; // they kept typing
    recipientInfo = r;
    const note = $("recipient-note");
    if (r.finalonly) {
      note.className = "card warn";
      note.innerHTML = `<strong>This address only accepts final payments.</strong> Your coins will land instantly and there is <u>no undo</u>. Check the address twice.`;
      note.hidden = false;
      $("window-seg").querySelectorAll("button").forEach((x) => (x.disabled = true));
    } else {
      $("window-seg").querySelectorAll("button").forEach((x) => (x.disabled = false));
      if (r.sent >= 5 && r.cancelled / r.sent >= 0.5) {
        note.className = "card warn";
        note.innerHTML = `<strong>Careful.</strong> This address has started ${r.sent} payments and taken back ${r.cancelled} of them. If they're paying <em>you</em>, wait for the countdown before handing anything over.`;
        note.hidden = false;
      } else if (r.sent > 0) {
        note.className = "card";
        note.innerHTML = `<span class="muted">This address has sent ${r.sent} undoable payment${r.sent === 1 ? "" : "s"} and taken back ${r.cancelled}.</span>`;
        note.hidden = false;
      }
    }
  } catch { /* the note is a courtesy; sending still works without it */ }
};

$("btn-send-go").onclick = async (e) => {
  const to = $("send-to").value.trim();
  const amt = parseAmount($("send-amount").value);
  if (!isAddress(to)) return toast("That isn't a Trefoil address.", true);
  if (to === S.address) return toast("That's your own address.", true);
  if (!amt || amt <= 0n) return toast("Enter an amount, e.g. 10 or 2.5", true);

  const finalOnly = !!recipientInfo?.finalonly;
  const window = finalOnly ? 0 : sendWindow;
  const ask = finalOnly
    ? `Send ${fmt(amt)} TFL to ${short(to)} now? This address only takes final payments — there is NO undo.`
    : `Send ${fmt(amt)} TFL to ${short(to)}? You can undo it for ${countdownWords(window)}.`;
  if (!confirm(ask)) return;

  busy(e.target, true, "Sending…");
  try {
    await sendTx(S.wallet, [{
      typeUrl: MsgDelayedSend.typeUrl,
      value: { creator: S.address, recipient: to, amount: { denom: CONFIG.denom, amount: amt.toString() }, window },
    }], { onStatus: (s) => (e.target.textContent = s) });
    toast(finalOnly ? `Sent ${fmt(amt)} TFL — final.` : `Sent ${fmt(amt)} TFL. You can undo it from Home.`);
    enterHome();
  } catch (err) { toast(err.message, true); }
  finally { busy(e.target, false, "Send"); }
};

function countdownWords(sec) {
  if (sec >= 3600) return `${sec / 3600} hour${sec >= 7200 ? "s" : ""}`;
  return `${sec / 60} minutes`;
}

// --- Receive ---------------------------------------------------------------------------

$("btn-receive").onclick = async () => {
  $("receive-addr").textContent = S.address;
  await QRCode.toCanvas($("qr"), S.address, { width: 220, margin: 1 });
  $("receive-note").textContent = S.record?.finalonly
    ? "Final-only is on: payments to you land instantly and can't be undone."
    : "Payments to you show as promised until their countdown ends, then they're yours.";
  show("receive");
};
$("btn-receive-back").onclick = () => show("home");
$("btn-copy-addr").onclick = () => copy(S.address);
$("home-addr").onclick = () => copy(S.address);

// --- Settings -------------------------------------------------------------------------

$("btn-settings").onclick = () => {
  $("set-rest").value = CONFIG.rest; $("set-chain").value = CONFIG.chainId;
  $("set-finalonly").checked = !!S.record?.finalonly;
  const r = S.record || { sent: 0, cancelled: 0 };
  $("my-record").textContent = r.sent === 0 ? "No undoable payments sent yet." : `${r.sent} sent, ${r.cancelled} taken back.`;
  show("settings");
};
$("btn-settings-back").onclick = () => show("home");

$("set-finalonly").onchange = async (e) => {
  const on = e.target.checked;
  const ask = on
    ? "Turn on final-only? People paying you will have no undo, and you can't be sent undoable payments."
    : "Turn off final-only? Payments to you will wait their countdown again.";
  if (!confirm(ask)) { e.target.checked = !on; return; }
  e.target.disabled = true;
  try {
    await sendTx(S.wallet, [{ typeUrl: MsgSetFinalOnly.typeUrl, value: { creator: S.address, finalonly: on } }]);
    toast(on ? "Final-only is on." : "Final-only is off.");
    await refresh();
  } catch (err) {
    toast(err.message, true);
    e.target.checked = !on;
  } finally { e.target.disabled = false; }
};

$("btn-settings-save").onclick = () => {
  const rest = $("set-rest").value.trim().replace(/\/$/, "");
  if (!/^https:\/\//.test(rest) && !/^http:\/\/(localhost|127\.0\.0\.1)/.test(rest)) {
    return toast("Only https is allowed for anything that isn't localhost.", true);
  }
  localStorage.setItem("trefoil.rest", rest);
  localStorage.setItem("trefoil.chainId", $("set-chain").value.trim());
  location.reload();
};
$("btn-forget").onclick = () => {
  if (!confirm("Remove this wallet from this browser? You'll need your 12-word phrase to get it back.")) return;
  localStorage.removeItem(STORE);
  location.reload();
};

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------

(async function start() {
  getParams().then((p) => (S.params = p)).catch(() => {});
  const saved = load();
  if (saved?.mnemonic) {
    S.wallet = await walletFromMnemonic(saved.mnemonic);
    S.address = await addressOf(S.wallet);
    enterHome();
  } else {
    show("welcome");
  }
  latestHeight().then((h) => { if (h) $("welcome-status").textContent = `Chain online · block ${h}`; })
    .catch(() => { $("welcome-status").textContent = `Can't reach the chain at ${CONFIG.rest}`; });
})();
