import { useEffect, useRef, useState, useCallback } from "react";
import { TOKEN_KEYS, getActiveRole } from "@/lib/token";

const BASE_URL = import.meta.env.VITE_BACKEND_URL as string;

function getWsUrl(studentId: number, token: string): string {
  const httpBase = BASE_URL.replace(/\/$/, "");
  const wsBase = httpBase.replace(/^http/, "ws");
  return `${wsBase}/view/students/${studentId}/live?token=${encodeURIComponent(token)}`;
}

function getToken(): string | null {
  const role = getActiveRole();
  if (!role) return null;
  return localStorage.getItem(TOKEN_KEYS[role]);
}

interface UseLiveVideoResult {
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  connected: boolean;
  error: string | null;
  reconnect: () => void;
}

export function useLiveVideo(studentId: number | null): UseLiveVideoResult {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mountedRef = useRef(true);

  const cleanup = useCallback(() => {
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
  }, []);

  const connect = useCallback(() => {
    cleanup();

    if (!studentId || !mountedRef.current) return;

    const token = getToken();
    if (!token) {
      setError("Not authenticated");
      return;
    }

    const url = getWsUrl(studentId, token);
    let ws: WebSocket;
    try {
      ws = new WebSocket(url);
    } catch {
      if (mountedRef.current) {
        setError("Failed to create WebSocket connection");
      }
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
        if (ctx) {
          ctx.drawImage(bitmap, 0, 0);
        }
        bitmap.close();
      } catch {
        // Frame decode failure — skip, next frame will arrive in ~1s
      }
    };

    ws.onerror = () => {
      if (!mountedRef.current) return;
      setError("WebSocket connection error");
    };

    ws.onclose = (event) => {
      if (!mountedRef.current) return;
      setConnected(false);

      if (!event.wasClean) {
        setError("Connection lost, reconnecting...");
        reconnectTimer.current = setTimeout(() => {
          if (mountedRef.current) {
            // eslint-disable-next-line react-hooks/immutability
            connect();
          }
        }, 3000);
      }
    };
  }, [studentId, cleanup]);

  useEffect(() => {
    mountedRef.current = true;
    connect();

    return () => {
      mountedRef.current = false;
      cleanup();
    };
  }, [connect, cleanup]);

  const reconnect = useCallback(() => {
    setError(null);
    connect();
  }, [connect]);

  return { canvasRef, connected, error, reconnect };
}
