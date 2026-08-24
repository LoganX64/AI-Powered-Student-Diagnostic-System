import { useEffect, useRef, useState, useCallback } from "react";
import { TOKEN_KEYS, getActiveRole } from "@/lib/token";

const BASE_URL = import.meta.env.VITE_BACKEND_URL as string;

function getToken(): string | null {
  const role = getActiveRole();
  if (!role) return null;
  return localStorage.getItem(TOKEN_KEYS[role]);
}

interface UseLiveVideoResult {
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  connected: boolean;
  live: boolean;
  error: string | null;
  reconnect: () => void;
}

export function useLiveVideo(studentId: number | null): UseLiveVideoResult {
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
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      wsRef.current.onmessage = null;
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnected(false);
  }, []);

  const checkLiveStatus = useCallback(async (id: number): Promise<boolean> => {
    const token = getToken();
    if (!token) return false;
    try {
      const httpBase = BASE_URL.replace(/\/$/, "");
      const res = await fetch(`${httpBase}/view/students/${id}/live/status?token=${encodeURIComponent(token)}`);
      if (!res.ok) return false;
      const data = await res.json();
      return data.live === true;
    } catch {
      return false;
    }
  }, []);

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
  }, [closeWs]);

  useEffect(() => {
    connectWsRef.current = connectWs;
  }, [connectWs]);

  const pollAndConnect = useCallback(async (id: number) => {
    const isLive = await checkLiveStatus(id);
    if (!mountedRef.current) return;

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
