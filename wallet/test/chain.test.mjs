// Checks that the wallet's encoders and address derivation match the chain.
// Run: node test/chain.test.mjs
import assert from "node:assert/strict";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { toHex } from "@cosmjs/encoding";

globalThis.localStorage = { getItem: () => null, setItem() {}, removeItem() {} };
const chain = await import("../src/chain.js");

// 1. Address prefix and shape
const w = await DirectSecp256k1HdWallet.generate(12, { prefix: "trefoil" });
const addr = await chain.addressOf(w);
assert.match(addr, /^trefoil1[02-9ac-hj-np-z]{38}$/);
assert.ok(chain.isAddress(addr));
assert.ok(!chain.isAddress("cosmos1abc"));

// 2. Same mnemonic -> same address (restore works)
const w2 = await chain.walletFromMnemonic(w.mnemonic);
assert.equal(await chain.addressOf(w2), addr);

// 3. MsgDelayedSend encoding round-trips with the chain's field numbers:
//    1 creator, 2 recipient, 3 amount (Coin: 1 denom, 2 amount), 4 window.
const msg = { creator: addr, recipient: "trefoil1bbb", amount: { denom: "utfl", amount: "100000000" }, window: 600 };
const bytes = chain.MsgDelayedSend.encode(msg).finish();
assert.equal(bytes[0], 0x0a, "creator should be field 1, wire type 2");
const hex = toHex(bytes);
assert.ok(hex.includes("12" + (11).toString(16).padStart(2, "0") + toHex(Buffer.from("trefoil1bbb"))), "recipient string present");
assert.ok(hex.includes("0a04" + toHex(Buffer.from("utfl"))), "coin denom nested in field 3");
assert.ok(hex.endsWith("20d804"), "window 600 as varint in field 4");
assert.deepEqual(chain.MsgDelayedSend.decode(bytes), msg);

// 3b. Cancel and final-only
const c = chain.MsgCancelSend.encode({ creator: addr, id: 7 }).finish();
assert.ok(toHex(c).endsWith("1007"), "id 7 as varint in field 2");
assert.deepEqual(chain.MsgCancelSend.decode(c), { creator: addr, id: 7 });
const f = chain.MsgSetFinalOnly.encode({ creator: addr, finalonly: true }).finish();
assert.ok(toHex(f).endsWith("1001"), "finalonly true in field 2");
assert.deepEqual(chain.MsgSetFinalOnly.decode(f), { creator: addr, finalonly: true });

// 3c. Countdown text
assert.equal(chain.countdown(521), "8:41");
assert.equal(chain.countdown(3780), "1h 03m");
assert.equal(chain.countdown(-5), "0:00");

// 4. Amount parsing / formatting
assert.equal(chain.parseAmount("10"), 10_000_000n);
assert.equal(chain.parseAmount("2.5"), 2_500_000n);
assert.equal(chain.parseAmount("0.000001"), 1n);
assert.equal(chain.parseAmount("abc"), null);
assert.equal(chain.fmt(5_000_000_000n), "5,000");
assert.equal(chain.fmt(1_234_560n), "1.23456");

console.log("ok — address derivation, undo message encoding, amounts, countdown");
