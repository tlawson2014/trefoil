# Trefoil Wallet — Screen Flow (v1, undo)

## Principles

Four rules every screen follows. If a screen breaks one, the screen is wrong.

1. The key never leaves the device. The wallet signs on your phone or laptop and sends only the signed bytes.
2. Pending is never money. An incoming payment that can still be undone is shown as a promise, in grey, outside the balance, with the words *don't hand anything over yet*.
3. Every outgoing send shows its countdown and an Undo button until it is final. When the countdown hits zero the button disappears and the word *Final* appears.
4. Before you send, the wallet tells you what it knows about the recipient: whether they are final-only (no undo), and how many of their own sends they have cancelled.

## Screens

### 1. Welcome
"The coin with an undo button." — *Create a wallet* / *I already have one*. Chain status at the bottom.

### 2. Create → your 12 words
Show, copy (numbered), confirm three words. Then straight to Home. No guardians, no extra accounts.

### 3. Home
- Balance (final coins only).
- **Outgoing pending**: each one a row — *Sent 5 TFL to …k2uk · arriving in 8:41 · [Undo]*. Live countdown.
- **Incoming pending**: grey rows — *5 TFL promised by …lzes · yours in 8:41 · don't hand anything over yet*.
- Recent finals below.
- Buttons: Send, Receive, Settings.

### 4. Send
Address, amount, and an undo window picker: **2 min · 10 min (default) · 1 hour · 24 hours**. Before the confirm button:
- If the recipient is final-only: "This address only accepts final payments. There is no undo. [I understand, pay now]" — sends with window 0.
- If the recipient has cancelled a notable share of their own sends: a warning line with the numbers.
On confirm: sign → send → the new row appears on Home with its countdown.

### 5. Undo
Tap Undo on a pending row → "Take back 5 TFL from …k2uk?" → sign → the row disappears, balance restored. Undo is refused (politely) if the countdown reached zero while the sheet was open.

### 6. Receive
Address as text and QR. If final-only is on, the screen says so.

### 7. Settings
- Node address (https enforced off localhost).
- **I sell things — accept final payments only.** Toggle → signs `set-final-only`. Explains the trade-off in one sentence.
- Show my undo record (sent / cancelled).
- Forget this wallet.

## Deferred
- Password-protected key storage (currently plain localStorage — prototype only).
- Address book / names for recipients.
- Push or email alert when a pending payment to you goes final.
- Guardian recovery as an optional extra (shelved, in git history).
