# Trefoil Wallet

A web wallet for Trefoil: one page, no install, works on a phone. The private
key never leaves the browser — the page signs on your device and sends only the
signed bytes to a node.

**What's in v1**

- Create a wallet (12 words, three-word check) or restore one
- Balance, send, receive (QR)
- Every send has an undo window you pick (2 min / 10 min / 1 h / 24 h) and
  shows on Home with a live countdown and an **Undo** button until it's final
- Incoming payments show as *promised — not yours yet* with their own
  countdown, outside the balance, with a warning not to hand anything over
- Before you send, the wallet tells you if the recipient is final-only (no
  undo) and how many of their own payments they've taken back
- The address box turns green when an address is valid and red with a reason
  when it isn't; addresses you've sent to before appear as tap-to-fill chips
- Home lists recent payments and ones you took back, in their own boxes
- Settings: the shop switch (*I sell things — final payments only*), your own
  public undo record, and your address book (tap an address to copy it)

**Prototype limitations** (fix before anyone uses it for real):

- The recovery phrase is stored in browser storage as-is. v1.1 adds a
  password.
- No push/email alerts; you see a payment go final only when the wallet is
  open.
- Pending sends are found by scanning the chain's whole pending table, which
  is fine for a test chain and not for a busy one.
- The address book lives in this browser only.
- History is read back from the node's own transaction index, so it follows
  your phrase to a new device. A node with that index switched off, or one
  that has pruned old blocks, falls back to the copy in this browser — the
  screen says which it is.

## Run it against your local chain

1. Start the chain: `ignite chain serve` in the repo root.
2. In a second terminal:
   ```bash
   cd wallet && python3 -m http.server 8080
   ```
3. Open http://localhost:8080 in a browser.

The wallet talks to the node's REST API at `http://localhost:1317`, which
`ignite chain serve` opens with CORS enabled. Change the API address under
Settings; anything that isn't localhost must be `https`.

To try undo with yourself as both sides, use two browser profiles (or one
normal window plus one private window) — each holds its own wallet. Fund the
first from the CLI:

```bash
trefoild --keyring-backend test tx bank send alice <wallet address> 1000000000utfl -y
```

## Develop

```bash
npm install
npm run build      # bundles src/ into dist/app.js (committed, so users need no build)
node test/chain.test.mjs
```

`src/chain.js` is everything that touches the chain (keys, signing, REST).
`src/app.js` is the screens. Wording follows `docs/WALLET-FLOW.md` in the repo.
