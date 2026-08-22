import { useEffect, useRef, useState, useCallback } from "react";
import { AlertCircle, Play, Loader2, Film } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

interface VideoChunksResponse {
  assignment_id: number;
  attempt_id: number;
  chunks: string[];
}

export function RecordedVideoPlayer({
  assignmentId,
  title,
}: {
  assignmentId: number;
  title?: string;
}) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [chunks, setChunks] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentIdx, setCurrentIdx] = useState(0);
  const [playing, setPlaying] = useState(false);
  const blobsRef = useRef<Map<string, string>>(new Map());

  const fetchChunks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<VideoChunksResponse>(
        `/admin/assignments/${assignmentId}/video-chunks`
      );
      setChunks(data.chunks || []);
      setCurrentIdx(0);
    } catch (e) {
      setError((e as Error).message || "Failed to load video chunks");
    } finally {
      setLoading(false);
    }
  }, [assignmentId]);

  useEffect(() => {
    fetchChunks();
    return () => {
      for (const url of blobsRef.current.values()) {
        URL.revokeObjectURL(url);
      }
      blobsRef.current.clear();
    };
  }, [fetchChunks]);

  const getChunkBlobUrl = useCallback(
    async (index: string): Promise<string | null> => {
      if (blobsRef.current.has(index)) {
        return blobsRef.current.get(index)!;
      }
      try {
        const res = await fetch(
          `${import.meta.env.VITE_BACKEND_URL}/admin/assignments/${assignmentId}/video-chunk/${index}`,
          {
            headers: {
              Authorization: `Bearer ${localStorage.getItem("admin_token") || localStorage.getItem("coach_token") || ""}`,
            },
          }
        );
        if (!res.ok) {
          return null;
        }
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        blobsRef.current.set(index, url);
        return url;
      } catch {
        return null;
      }
    },
    [assignmentId]
  );

  const playChunk = useCallback(
    async (idx: number) => {
      if (idx >= chunks.length || !videoRef.current) {
        setPlaying(false);
        return;
      }

      const chunkIndex = chunks[idx];
      const blobUrl = await getChunkBlobUrl(chunkIndex);
      if (!blobUrl) {
        setError(`Failed to load chunk ${chunkIndex}`);
        setPlaying(false);
        return;
      }

      const video = videoRef.current;
      video.src = blobUrl;
      video.onended = () => {
        setCurrentIdx(idx + 1);
        playChunk(idx + 1);
      };
      video.onerror = () => {
        setError(`Playback error on chunk ${chunkIndex}`);
        setPlaying(false);
      };

      try {
        await video.play();
        setPlaying(true);
        setError(null);
      } catch {
        setError("Playback failed");
        setPlaying(false);
      }
    },
    [chunks, getChunkBlobUrl]
  );

  const handlePlay = () => {
    if (chunks.length === 0) return;
    setError(null);
    playChunk(currentIdx);
  };

  const handlePause = () => {
    if (videoRef.current) {
      videoRef.current.pause();
    }
    setPlaying(false);
  };

  const handleReset = () => {
    if (videoRef.current) {
      videoRef.current.pause();
      videoRef.current.currentTime = 0;
    }
    setCurrentIdx(0);
    setPlaying(false);
  };

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="ml-2 text-sm text-muted-foreground">
            Loading video...
          </span>
        </CardContent>
      </Card>
    );
  }

  if (error && chunks.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center gap-2 py-8">
          <AlertCircle className="h-5 w-5 text-destructive" />
          <span className="text-sm text-destructive">{error}</span>
          <Button variant="outline" size="sm" onClick={fetchChunks}>
            Retry
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <Film className="h-4 w-4" />
          {title || "Recorded Video"}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="relative">
          <video
            ref={videoRef}
            className="w-full max-w-[640px] rounded-lg border bg-black"
            controls={false}
          />
          {chunks.length === 0 && (
            <div className="absolute inset-0 flex items-center justify-center text-muted-foreground text-sm">
              No recorded footage available
            </div>
          )}
        </div>
        {error && (
          <div className="mt-2 flex items-center gap-1 text-sm text-destructive">
            <AlertCircle className="h-3 w-3" />
            {error}
          </div>
        )}
        {chunks.length > 0 && (
          <div className="mt-2 flex items-center gap-2">
            {!playing ? (
              <Button
                variant="outline"
                size="sm"
                onClick={handlePlay}
                className="gap-1"
              >
                <Play className="h-3 w-3" />
                {currentIdx === 0 ? "Play" : "Resume"}
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                onClick={handlePause}
              >
                Pause
              </Button>
            )}
            {currentIdx > 0 && (
              <Button variant="ghost" size="sm" onClick={handleReset}>
                Reset
              </Button>
            )}
            <span className="text-xs text-muted-foreground">
              {currentIdx + 1} / {chunks.length} chunks
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
