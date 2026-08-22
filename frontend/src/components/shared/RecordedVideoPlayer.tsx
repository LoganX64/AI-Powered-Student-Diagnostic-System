import { useEffect, useRef, useState, useCallback } from "react";
import { AlertCircle, Loader2, Film, Trash2, RefreshCw } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { deleteVideo, mergeVideo } from "@/services/dashboard.service";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

interface VideoChunksResponse {
  assignment_id: number;
  attempt_id: number;
  chunks: string[];
  has_merged: boolean;
}

export function RecordedVideoPlayer({
  assignmentId,
  title,
  onDelete,
}: {
  assignmentId: number;
  title?: string;
  onDelete?: () => void;
}) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hasMerged, setHasMerged] = useState(false);
  const [merging, setMerging] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchChunks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<VideoChunksResponse>(
        `/admin/assignments/${assignmentId}/video-chunks`
      );
      setHasMerged(data.has_merged);
      if (!data.has_merged && data.chunks.length > 0) {
        startPolling();
      }
    } catch (e) {
      setError((e as Error).message || "Failed to load video");
    } finally {
      setLoading(false);
    }
  }, [assignmentId]);

  const startPolling = useCallback(() => {
    if (pollRef.current) return;
    pollRef.current = setInterval(async () => {
      try {
        const data = await apiFetch<VideoChunksResponse>(
          `/admin/assignments/${assignmentId}/video-chunks`
        );
        if (data.has_merged) {
          setHasMerged(true);
          if (pollRef.current) {
            clearInterval(pollRef.current);
            pollRef.current = null;
          }
        }
      } catch {
        // ignore polling errors
      }
    }, 5000);
  }, [assignmentId]);

  useEffect(() => {
    fetchChunks();
    return () => {
      if (pollRef.current) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
    };
  }, [fetchChunks]);

  const handleMerge = async () => {
    setMerging(true);
    try {
      await mergeVideo(assignmentId);
      setHasMerged(true);
      if (pollRef.current) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
    } catch (e) {
      setError((e as Error).message || "Failed to merge video");
    } finally {
      setMerging(false);
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await deleteVideo(assignmentId);
      setHasMerged(false);
      if (videoRef.current) {
        videoRef.current.src = "";
      }
      if (onDelete) {
        onDelete();
      }
    } catch (e) {
      setError((e as Error).message || "Failed to delete video");
    } finally {
      setDeleting(false);
    }
  };

  const videoUrl = hasMerged
    ? `${import.meta.env.VITE_BACKEND_URL}/admin/assignments/${assignmentId}/video-merged`
    : "";

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

  if (error && !hasMerged) {
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
        <CardTitle className="flex items-center justify-between text-sm">
          <div className="flex items-center gap-2">
            <Film className="h-4 w-4" />
            {title || "Recorded Video"}
          </div>
          {hasMerged && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="ghost" size="sm" className="gap-1 text-destructive hover:text-destructive">
                  <Trash2 className="h-3 w-3" />
                  Delete
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete Video Permanently?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will permanently delete the recorded video for this exam.
                    This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handleDelete}
                    disabled={deleting}
                    className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  >
                    {deleting ? "Deleting..." : "Delete Permanently"}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="relative">
          {hasMerged ? (
            <video
              ref={videoRef}
              src={videoUrl}
              controls
              className="w-full max-w-[640px] rounded-lg border bg-black"
            />
          ) : (
            <div className="flex flex-col items-center justify-center gap-3 py-8 text-muted-foreground">
              {merging ? (
                <>
                  <Loader2 className="h-5 w-5 animate-spin" />
                  <span className="text-sm">Merging video chunks...</span>
                </>
              ) : (
                <>
                  <span className="text-sm">Video is being processed into a single file.</span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleMerge}
                    className="gap-1"
                  >
                    <RefreshCw className="h-3 w-3" />
                    Merge Now
                  </Button>
                </>
              )}
            </div>
          )}
        </div>
        {error && hasMerged && (
          <div className="mt-2 flex items-center gap-1 text-sm text-destructive">
            <AlertCircle className="h-3 w-3" />
            {error}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
