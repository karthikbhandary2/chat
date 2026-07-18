import { useState, useEffect, useRef, useCallback } from "react";

const API_BASE = "http://localhost:8082/api/v1";
const WS_BASE = "ws://localhost:8082";

// ---- small helpers -------------------------------------------------------

function colorForUser(id) {
  let hash = 0;
  for (let i = 0; i < id.length; i++) hash = id.charCodeAt(i) + ((hash << 5) - hash);
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue} 65% 62%)`;
}

function formatTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function formatDay(iso) {
  const d = new Date(iso);
  return d.toLocaleDateString([], { month: "short", day: "numeric" });
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
  const [mode, setMode] = useState("login");
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

// ---- sidebar: conversation list + start-new form ---------------------------

function Sidebar({ token, user, activeId, onSelect, refreshKey, onCreated }) {
  const [conversations, setConversations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showNewForm, setShowNewForm] = useState(false);
  const [username, setUsername] = useState("");
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);

  const loadConversations = useCallback(async () => {
    try {
      const res = await api("/conversations", { headers: { Authorization: `Bearer ${token}` } });
      setConversations(res || []);
    } catch (err) {
      console.error("failed to load conversations:", err);
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    loadConversations();
  }, [loadConversations, refreshKey]);

  const createConversation = async (e) => {
    e.preventDefault();
    setError("");
    if (!username.trim()) return;
    setCreating(true);
    try {
      const conv = await api("/conversations", {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: JSON.stringify({ type: "direct", usernames: [username.trim()] }),
      });
      setUsername("");
      setShowNewForm(false);
      await loadConversations();
      onCreated(conv.id);
    } catch (err) {
      setError(err.message);
    } finally {
      setCreating(false);
    }
  };

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <span className="sidebar-title">conversations</span>
        <button className="icon-btn" onClick={() => setShowNewForm((s) => !s)} title="start a conversation">
          {showNewForm ? "×" : "+"}
        </button>
      </div>

      {showNewForm && (
        <form className="new-convo-form" onSubmit={createConversation}>
          <input
            placeholder="username to message"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoFocus
          />
          <button type="submit" disabled={creating}>{creating ? "…" : "start"}</button>
          {error && <div className="sidebar-error">{error}</div>}
        </form>
      )}

      <div className="convo-list">
        {loading && <div className="sidebar-empty">loading…</div>}
        {!loading && conversations.length === 0 && (
          <div className="sidebar-empty">no conversations yet — start one above</div>
        )}
        {conversations.map((c) => (
          <button
            key={c.id}
            className={`convo-item ${c.id === activeId ? "active" : ""}`}
            onClick={() => onSelect(c.id)}
          >
            <span className={`convo-type-dot ${c.type}`} />
            <span className="convo-label">
              <span className="convo-id">{c.id.slice(0, 8)}</span>
              <span className="convo-type">{c.type}</span>
            </span>
            <span className="convo-date">{formatDay(c.created_at)}</span>
          </button>
        ))}
      </div>

      <div className="sidebar-footer">
        <span className="me-line">
          <span className="dot online pulse" />
          {user.username}
        </span>
      </div>
    </aside>
  );
}

// ---- main chat panel -------------------------------------------------------

function ChatPanel({ token, user, conversationId, onTypingEvent }) {
  const [messages, setMessages] = useState([]);
  const [draft, setDraft] = useState("");
  const [historyLoaded, setHistoryLoaded] = useState(false);
  const logRef = useRef(null);
  const wsRef = useRef(null);
  const typingTimeoutRef = useRef(null);

  const loadHistory = useCallback(async () => {
    if (!conversationId) return;
    setHistoryLoaded(false);
    try {
      const res = await api(`/conversations/${conversationId}/messages`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const list = (res || []).slice().reverse();
      setMessages(
        list.map((m) => ({
          id: m.id,
          senderId: m.sender_id,
          content: m.content,
          createdAt: m.created_at,
        }))
      );
    } catch (err) {
      console.error("failed to load history:", err);
    } finally {
      setHistoryLoaded(true);
    }
  }, [conversationId, token]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  // shared websocket connection, owned by the parent via a ref passed down
  useEffect(() => {
    const ws = new WebSocket(`${WS_BASE}/ws?token=${encodeURIComponent(token)}`);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      let parsed = null;
      try {
        parsed = JSON.parse(event.data);
      } catch (e) {
        // Shouldn't happen once the backend always sends JSON envelopes.
        // Left as a visible signal rather than silently swallowing it,
        // so a backend regression is obvious instead of showing as
        // mysterious missing messages.
        console.warn("received non-JSON websocket payload:", event.data);
        return;
      }

      if (parsed.type === "typing") {
        onTypingEvent(parsed.user_id);
        return;
      }

      if (parsed.type === "message") {
        // Our own messages are already rendered optimistically in send();
        // this is just the hub echoing them back to our own connection.
        if (parsed.sender_id === user.id) return;

        setMessages((prev) => [
          ...prev,
          {
            id: parsed.id,
            senderId: parsed.sender_id,
            content: parsed.content,
            createdAt: parsed.created_at,
          },
        ]);
      }
    };

    return () => ws.close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  useEffect(() => {
    logRef.current?.scrollTo({ top: logRef.current.scrollHeight, behavior: "smooth" });
  }, [messages]);

  const send = () => {
    const text = draft.trim();
    if (!text || !conversationId || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(JSON.stringify({ type: "message", conversation_id: conversationId, content: text }));
    setMessages((prev) => [
      ...prev,
      { id: `local-${Date.now()}`, senderId: user.id, content: text, createdAt: new Date().toISOString(), pending: true },
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

  if (!conversationId) {
    return (
      <div className="chat-panel">
        <div className="log-empty centered">select a conversation, or start a new one</div>
      </div>
    );
  }

  return (
    <div className="chat-panel">
      <main className="log" ref={logRef}>
        {historyLoaded && messages.length === 0 && <div className="log-empty">no messages yet — say something</div>}
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
      </main>

      <footer className="composer">
        <input
          value={draft}
          onChange={onDraftChange}
          onKeyDown={(e) => e.key === "Enter" && send()}
          placeholder="write a message"
        />
        <button onClick={send} disabled={!draft.trim()}>send</button>
      </footer>
    </div>
  );
}

// ---- chat screen: sidebar + panel, shared connection/typing state -----------

function ChatScreen({ token, user, onLogout }) {
  const [conversationId, setConversationId] = useState(null);
  const [connected, setConnected] = useState(true); // panel owns its own socket; kept simple here
  const [typingUsers, setTypingUsers] = useState({});
  const [refreshKey, setRefreshKey] = useState(0);

  const handleTypingEvent = (userId) => {
    if (userId === user.id) return;
    setTypingUsers((prev) => ({ ...prev, [userId]: Date.now() }));
    setTimeout(() => {
      setTypingUsers((prev) => {
        if (Date.now() - (prev[userId] || 0) < 2500) return prev;
        const next = { ...prev };
        delete next[userId];
        return next;
      });
    }, 3000);
  };

  const typingList = Object.keys(typingUsers);

  return (
    <div className="chat-screen">
      <header className="topbar">
        <div className="topbar-left">
          <span className="dot pulse online" />
          <span className="brand-text">relay</span>
        </div>
        <div className="topbar-right">
          <button className="ghost-btn" onClick={onLogout}>sign out</button>
        </div>
      </header>

      <div className="chat-body">
        <Sidebar
          token={token}
          user={user}
          activeId={conversationId}
          onSelect={setConversationId}
          refreshKey={refreshKey}
          onCreated={(id) => {
            setConversationId(id);
            setRefreshKey((k) => k + 1);
          }}
        />
        <div className="chat-main">
          <ChatPanel
            token={token}
            user={user}
            conversationId={conversationId}
            onTypingEvent={handleTypingEvent}
          />
          {typingList.length > 0 && (
            <div className="typing-indicator floating">
              {typingList.map((id) => id.slice(0, 8)).join(", ")} typing
              <span className="typing-dots"><i /><i /><i /></span>
            </div>
          )}
        </div>
      </div>
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