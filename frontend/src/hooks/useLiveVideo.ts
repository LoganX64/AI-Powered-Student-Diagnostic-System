import { useEffect, useRef, useState, useCallback } from "react";

const BASE_URL = import.meta.env.VITE_BACKEND_URL as string;

interface UseLiveVideoResult {
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  connected: boolean;
  live: boolean;
  error: string | null;
  reconnect: () => void;
}

export function useLiveVideo(
  studentId: number | null,
  tokenKey: string,
): UseLiveVideoResult {
  const getToken = useCallback(
    () => localStorage.getItem(tokenKey),
    [tokenKey],
  );
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [connected, setConnected] = useState(false);
  const [live, setLive] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const connectWsRef = useRef<(id: number) => void>(() => {});
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const statusTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  const mountedRef = useRef(true);

  const cleanup = useCallback(() => {
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current);
      reconnectTimer.current = null;
    }
    if (statusTimer.current) {
      clearInterval(statusTimer.current);
      statusTimer.current = null;
    }
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      wsRef.current.onmessage = null;
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  const closeWs = useCallback(() => {
    // A pending reconnect must be cancelled whenever the socket is torn down.
    // Otherwise the 5s status poll can reconnect first and the stale 3s timer
    // then closes that healthy socket to open a second connection.
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current);
      reconnectTimer.current = null;
    }
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      wsRef.current.onmessage = null;
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnected(false);
  }, []);

  /**
   * `true` live, `false` definitely not live, `null` could not determine.
   * Collapsing every failure into `false` made a 401/403/5xx or a dropped
   * connection indistinguishable from an idle student, so the panel reported
   * "not currently live" during an outage.
   */
  const checkLiveStatus = useCallback(async (id: number): Promise<boolean | null> => {
    const token = getToken();
    if (!token) return null;
    try {
      const httpBase = BASE_URL.replace(/\/$/, "");
      const res = await fetch(`${httpBase}/view/students/${id}/live/status?token=${encodeURIComponent(token)}`);
      if (!res.ok) return null;
      const data = await res.json();
      return data.live === true;
    } catch {
      return null;
    }
  }, [getToken]);

  const connectWs = useCallback((id: number) => {
    closeWs();

    const token = getToken();
    if (!token || !mountedRef.current) return;

    const httpBase = BASE_URL.replace(/\/$/, "");
    const wsBase = httpBase.replace(/^http/, "ws");
    const url = `${wsBase}/view/students/${id}/live?token=${encodeURIComponent(token)}`;

    let ws: WebSocket;
    try {
      ws = new WebSocket(url);
    } catch {
      if (mountedRef.current) setError("Failed to create WebSocket connection");
      return;
    }

    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    ws.onopen = () => {
      if (!mountedRef.current) return;
      setConnected(true);
      setError(null);
    };

    ws.onmessage = async (event: MessageEvent) => {
      if (!mountedRef.current) return;
      const canvas = canvasRef.current;
      if (!canvas) return;

      try {
        const blob = new Blob([event.data], { type: "image/jpeg" });
        const bitmap = await createImageBitmap(blob);
        canvas.width = bitmap.width;
        canvas.height = bitmap.height;
        const ctx = canvas.getContext("2d");
        if (ctx) ctx.drawImage(bitmap, 0, 0);
        bitmap.close();
      } catch {
        // Frame decode failure — skip, next frame will arrive in ~1s
      }
    };

    ws.onerror = () => {
      if (!mountedRef.current) return;
    };

    ws.onclose = (event) => {
      if (!mountedRef.current) return;
      setConnected(false);

      if (!event.wasClean && mountedRef.current) {
        setError("Connection lost, reconnecting...");
        reconnectTimer.current = setTimeout(() => {
          if (mountedRef.current) connectWsRef.current(id);
        }, 3000);
      }
    };
  }, [closeWs, getToken]);

  useEffect(() => {
    connectWsRef.current = connectWs;
  }, [connectWs]);

  const pollAndConnect = useCallback(async (id: number) => {
    const isLive = await checkLiveStatus(id);
    if (!mountedRef.current) return;

    if (isLive === null) {
      // The check itself failed. Say so — do not report the student as idle.
      setLive(false);
      closeWs();
      setError("Could not determine live status — check your connection.");
      return;
    }

    setLive(isLive);
    if (isLive) {
      setError(null);
      if (!wsRef.current || wsRef.current.readyState === WebSocket.CLOSED) {
        connectWs(id);
      }
    } else {
      closeWs();
      setError(null);
    }
  }, [checkLiveStatus, connectWs, closeWs]);

  useEffect(() => {
    mountedRef.current = true;
    if (!studentId) return;

    pollAndConnect(studentId);
    statusTimer.current = setInterval(() => {
      if (mountedRef.current) pollAndConnect(studentId);
    }, 5000);

    return () => {
      mountedRef.current = false;
      cleanup();
    };
  }, [studentId, pollAndConnect, cleanup]);

  const reconnect = useCallback(() => {
    setError(null);
    if (studentId) pollAndConnect(studentId);
  }, [studentId, pollAndConnect]);

  return { canvasRef, connected, live, error, reconnect };
}
