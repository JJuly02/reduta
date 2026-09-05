import { useEffect, useRef } from "react";

// useEventWS subscribes to realtime messages for an event and invokes onType
// with the message type ("scoreboard" | "challenges.changed" | ...).
export function useEventWS(eventId: string | undefined, onType: (t: string, data?: any) => void) {
  const cb = useRef(onType);
  cb.current = onType;
  useEffect(() => {
    if (!eventId) return;
    let closed = false;
    let ws: WebSocket | null = null;
    let timer: number | undefined;
    const connect = () => {
      if (closed) return;
      const proto = location.protocol === "https:" ? "wss" : "ws";
      ws = new WebSocket(`${proto}://${location.host}/ws?event=${encodeURIComponent(eventId)}`);
      ws.onmessage = (e) => {
        try {
          const m = JSON.parse(e.data);
          if (m && m.type) cb.current(m.type, m.data);
        } catch {
          /* ignore */
        }
      };
      ws.onclose = () => {
        if (!closed) timer = window.setTimeout(connect, 2000);
      };
      ws.onerror = () => ws?.close();
    };
    connect();
    return () => {
      closed = true;
      if (timer) clearTimeout(timer);
      ws?.close();
    };
  }, [eventId]);
}
