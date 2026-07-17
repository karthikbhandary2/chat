import { useState, useEffect, useRef, useCallback } from "react";

const API_BASE = "http://localhost:8082/api/v1";
const WS_BASE = "ws://localhost:8082";

// ---- small helpers -------------------------------------------------------

function colorForUser(id) {
  // deterministic hue from the user id, so the same sender always gets the
  // same color across sessions without needing a lookup table
  let hash = 0;
  for (let i = 0; i < id.length; i++) hash = id.charCodeAt(i) + ((hash << 5) - hash);
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue} 65% 62%)`;
}

function formatTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

async function api(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });
  const text = await res.text();
  let body = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = text;
  }
  if (!res.ok) {
    const message = (body && body.error) || (typeof body === "string" ? body : `Request failed (${res.status})`);
    throw new Error(message);
  }
  return body;
}

// ---- auth screen ----------------------------------------------------------

function AuthScreen({ onAuthenticated }) {
  const [mode, setMode] = useState("login"); // "login" | "register"
  const [form, setForm] = useState({ username: "", email: "", password: "", full_name: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const update = (field) => (e) => setForm((f) => ({ ...f, [field]: e.target.value }));

  const submit = async (e) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      if (mode === "register") {
        await api("/register", {
          method: "POST",
          body: JSON.stringify({
            username: form.username,
            email: form.email,
            password: form.password,
            full_name: form.full_name,
          }),
        });
      }
      const loginRes = await api("/login", {
        method: "POST",
        body: JSON.stringify({ email: form.email, password: form.password }),
      });
      onAuthenticated(loginRes.token, loginRes.user);
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-screen">
      <div className="auth-card">
        <div className="auth-brand">
          <span className="dot pulse" />
          <span className="auth-brand-text">relay</span>
        </div>
        <p className="auth-sub">
          {mode === "login" ? "sign in to resume your session" : "create an account to open a connection"}
        </p>

        <form onSubmit={submit} className="auth-form">
          {mode === "register" && (
            <>
              <label>username</label>
              <input value={form.username} onChange={update("username")} required minLength={3} />
              <label>full name</label>
              <input value={form.full_name} onChange={update("full_name")} required />
            </>
          )}
          <label>email</label>
          <input type="email" value={form.email} onChange={update("email")} required />
          <label>password</label>
          <input type="password" value={form.password} onChange={update("password")} required minLength={8} />

          {error && <div className="auth-error">{error}</div>}

          <button type="submit" disabled={busy}>
            {busy ? "working…" : mode === "login" ? "sign in" : "create account"}
          </button>
        </form>

        <button
          className="auth-switch"
          onClick={() => {
            setMode(mode === "login" ? "register" : "login");
            setError("");
          }}
        >
          {mode === "login" ? "need an account? register" : "have an account? sign in"}
        </button>
      </div>
    </div>
  );
}

// ---- main chat screen -------------------------------------------------------

function ChatScreen({ token, user, onLogout }) {
  const [conversationId, setConversationId] = useState(localStorage.getItem("relay_conv") || "");
  const [connected, setConnected] = useState(false);
  const [messages, setMessages] = useState([]);
  const [draft, setDraft] = useState("");
  const [typingUsers, setTypingUsers] = useState({}); // userId -> timeout handle marker
  const [presence, setPresence] = useState(null); // for a manually-checked peer id
  const [peerCheckId, setPeerCheckId] = useState("");
  const [historyLoaded, setHistoryLoaded] = useState(false);

  const wsRef = useRef(null);
  const logRef = useRef(null);
  const typingTimeoutRef = useRef(null);

  // persist the conversation id so a refresh doesn't lose it
  useEffect(() => {
    if (conversationId) localStorage.setItem("relay_conv", conversationId);
  }, [conversationId]);

  const loadHistory = useCallback(async () => {
    if (!conversationId) return;
    try {
      const res = await api(`/conversations/${conversationId}/messages`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const list = (res || []).slice().reverse();
      setMessages(
        list.map((m) => ({
          id: m.id,
          conversationId: m.conversation_id,
          senderId: m.sender_id,
          content: m.content,
          createdAt: m.created_at,
        }))
      );
      setHistoryLoaded(true);
    } catch (err) {
      console.error("failed to load history:", err);
      setHistoryLoaded(true);
    }
  }, [conversationId, token]);

  useEffect(() => {
    if (conversationId) loadHistory();
  }, [conversationId, loadHistory]);

  // ---- websocket lifecycle ----

  const connect = useCallback(() => {
    if (wsRef.current) wsRef.current.close();
    const ws = new WebSocket(`${WS_BASE}/ws?token=${encodeURIComponent(token)}`);

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);

    ws.onmessage = (event) => {
      let parsed = null;
      try {
        parsed = JSON.parse(event.data);
      } catch {
        // plain-text chat content (current server behavior for chat messages)
        setMessages((prev) => [
          ...prev,
          {
            id: `local-${Date.now()}`,
            conversationId,
            senderId: "unknown",
            content: event.data,
            createdAt: new Date().toISOString(),
          },
        ]);
        return;
      }

      if (parsed.type === "typing") {
        markTyping(parsed.user_id);
      }
    };

    wsRef.current = ws;
  }, [token, conversationId]);

  useEffect(() => {
    connect();
    return () => wsRef.current?.close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function markTyping(userId) {
    setTypingUsers((prev) => ({ ...prev, [userId]: Date.now() }));
    setTimeout(() => {
      setTypingUsers((prev) => {
        if (Date.now() - (prev[userId] || 0) < 2500) return prev;
        const next = { ...prev };
        delete next[userId];
        return next;
      });
    }, 3000);
  }

  useEffect(() => {
    logRef.current?.scrollTo({ top: logRef.current.scrollHeight, behavior: "smooth" });
  }, [messages, typingUsers]);

  const send = () => {
    const text = draft.trim();
    if (!text || !conversationId || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(
      JSON.stringify({ type: "message", conversation_id: conversationId, content: text })
    );
    setMessages((prev) => [
      ...prev,
      { id: `local-${Date.now()}`, conversationId, senderId: user.id, content: text, createdAt: new Date().toISOString(), pending: true },
    ]);
    setDraft("");
  };

  const onDraftChange = (e) => {
    setDraft(e.target.value);
    if (!conversationId || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    if (typingTimeoutRef.current) return;
    wsRef.current.send(JSON.stringify({ type: "typing", conversation_id: conversationId }));
    typingTimeoutRef.current = setTimeout(() => {
      typingTimeoutRef.current = null;
    }, 2000);
  };

  const checkPresence = async () => {
    if (!peerCheckId) return;
    try {
      const res = await api(`/users/${peerCheckId}/presence`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      setPresence(res.online);
    } catch (err) {
      setPresence(null);
      console.error(err);
    }
  };

  const typingList = Object.keys(typingUsers).filter((id) => id !== user.id);

  return (
    <div className="chat-screen">
      <header className="topbar">
        <div className="topbar-left">
          <span className={`dot ${connected ? "pulse online" : "offline"}`} />
          <span className="brand-text">relay</span>
          <span className="topbar-status">{connected ? "connected" : "disconnected"}</span>
        </div>
        <div className="topbar-right">
          <span className="me">{user.username}</span>
          <button className="ghost-btn" onClick={onLogout}>sign out</button>
        </div>
      </header>

      <div className="session-bar">
        <div className="session-field">
          <label>conversation</label>
          <input
            value={conversationId}
            onChange={(e) => setConversationId(e.target.value.trim())}
            placeholder="paste conversation id"
          />
        </div>
        <div className="session-field">
          <label>check presence</label>
          <div className="presence-row">
            <input value={peerCheckId} onChange={(e) => setPeerCheckId(e.target.value.trim())} placeholder="user id" />
            <button className="ghost-btn" onClick={checkPresence}>check</button>
            {presence !== null && (
              <span className={`presence-pill ${presence ? "online" : "offline"}`}>
                {presence ? "online" : "offline"}
              </span>
            )}
          </div>
        </div>
        <div className="session-field">
          <label>my id</label>
          <code className="my-id">{user.id}</code>
        </div>
      </div>

      <main className="log" ref={logRef}>
        {!conversationId && <div className="log-empty">enter a conversation id above to load messages</div>}
        {conversationId && historyLoaded && messages.length === 0 && (
          <div className="log-empty">no messages yet — say something</div>
        )}
        {messages.map((m) => {
          const mine = m.senderId === user.id;
          return (
            <div key={m.id} className={`log-line ${mine ? "mine" : ""}`}>
              <span className="log-time">{formatTime(m.createdAt)}</span>
              <span className="log-sender" style={{ color: colorForUser(m.senderId) }}>
                {mine ? "you" : m.senderId.slice(0, 8)}
              </span>
              <span className="log-content">{m.content}</span>
              {m.pending && <span className="log-pending">·</span>}
            </div>
          );
        })}
        {typingList.length > 0 && (
          <div className="typing-indicator">
            {typingList.map((id) => id.slice(0, 8)).join(", ")} typing
            <span className="typing-dots"><i /><i /><i /></span>
          </div>
        )}
      </main>

      <footer className="composer">
        <input
          value={draft}
          onChange={onDraftChange}
          onKeyDown={(e) => e.key === "Enter" && send()}
          placeholder={conversationId ? "write a message" : "set a conversation id first"}
          disabled={!conversationId}
        />
        <button onClick={send} disabled={!conversationId || !draft.trim()}>send</button>
      </footer>
    </div>
  );
}

// ---- root -------------------------------------------------------------------

export default function App() {
  const [session, setSession] = useState(() => {
    const token = localStorage.getItem("relay_token");
    const user = localStorage.getItem("relay_user");
    return token && user ? { token, user: JSON.parse(user) } : null;
  });

  const handleAuthenticated = (token, user) => {
    localStorage.setItem("relay_token", token);
    localStorage.setItem("relay_user", JSON.stringify(user));
    setSession({ token, user });
  };

  const handleLogout = () => {
    localStorage.removeItem("relay_token");
    localStorage.removeItem("relay_user");
    setSession(null);
  };

  return (
    <>
      {session ? (
        <ChatScreen token={session.token} user={session.user} onLogout={handleLogout} />
      ) : (
        <AuthScreen onAuthenticated={handleAuthenticated} />
      )}
    </>
  );
}