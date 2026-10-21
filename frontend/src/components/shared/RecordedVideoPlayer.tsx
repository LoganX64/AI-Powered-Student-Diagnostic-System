import { useEffect, useState, useCallback, useRef } from "react";
import { AlertCircle, Loader2, Film, Trash2, RefreshCw } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { currentPrefix } from "@/config/routes";
import { deleteVideo, getVideoToken } from "@/services/dashboard.service";
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

interface VideoStatusResponse {
  assignment_id: number;
  attempt_id: number;
  status?: "processing" | "ready" | "failed" | "none";
  has_merged?: boolean;
  chunks_count?: number;
  message?: string;
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
  const [hasVideo, setHasVideo] = useState<boolean | null>(null);
  const [videoToken, setVideoToken] = useState<string | null>(null);
  const [mergeStatus, setMergeStatus] = useState<"idle" | "processing" | "ready" | "failed">("idle");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const checkVideo = useCallback(async (retry = false) => {
    setError(null);
    try {
      const url = `${currentPrefix()}/assignments/${assignmentId}/video-status${retry ? "?retry=true" : ""}`;
      const data = await apiFetch<VideoStatusResponse>(url);

      if (data.status === "ready" || data.has_merged) {
        setHasVideo(true);
        setMergeStatus("ready");
        const tokenData = await getVideoToken(assignmentId);
        setVideoToken(tokenData.token);
      } else if (data.status === "processing") {
        setHasVideo(true);
        setMergeStatus("processing");
      } else if (data.status === "failed") {
        setHasVideo(true);
        setMergeStatus("failed");
        setError(data.message || "Video merge failed. You can retry.");
      } else {
        setHasVideo(false);
        setMergeStatus("idle");
      }
    } catch (e) {
      setHasVideo(false);
      setMergeStatus("idle");
      setError((e as Error).message || "Failed to load video");
    } finally {
      setLoading(false);
    }
  }, [assignmentId]);

  useEffect(() => {
    checkVideo();
    return () => {
      if (pollTimerRef.current) {
        clearTimeout(pollTimerRef.current);
      }
    };
  }, [checkVideo]);

  // Polling loop when status is processing
  useEffect(() => {
    if (mergeStatus !== "processing") return;

    pollTimerRef.current = setTimeout(() => {
      checkVideo();
    }, 3000);

    return () => {
      if (pollTimerRef.current) {
        clearTimeout(pollTimerRef.current);
      }
    };
  }, [mergeStatus, checkVideo]);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await deleteVideo(assignmentId);
      setHasVideo(false);
      setMergeStatus("idle");
      if (onDelete) {
        onDelete();
      }
    } catch (e) {
      setError((e as Error).message || "Failed to delete video");
    } finally {
      setDeleting(false);
    }
  };

  const videoUrl = videoToken
    ? `${import.meta.env.VITE_BACKEND_URL}${currentPrefix()}/assignments/${assignmentId}/video-merged?token=${videoToken}`
    : "";

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="ml-2 text-sm text-muted-foreground">
            Checking video recording...
          </span>
        </CardContent>
      </Card>
    );
  }

  if (!hasVideo) {
    return null;
  }

  if (mergeStatus === "processing") {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-sm">
            <Film className="h-4 w-4" />
            {title || "Recorded Video"}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center justify-center py-8 text-center">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
          <p className="mt-2 text-sm font-medium">Processing Proctoring Video...</p>
          <p className="mt-1 max-w-sm text-xs text-muted-foreground">
            Verifying and stitching recording segments. This usually completes in under a minute.
          </p>
        </CardContent>
      </Card>
    );
  }

  if (mergeStatus === "failed") {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between text-sm">
            <div className="flex items-center gap-2">
              <Film className="h-4 w-4" />
              {title || "Recorded Video"}
            </div>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center justify-center py-6 text-center">
          <AlertCircle className="h-6 w-6 text-destructive" />
          <p className="mt-2 text-sm font-medium text-destructive">Video Processing Failed</p>
          <p className="mt-1 text-xs text-muted-foreground">
            {error || "An error occurred while merging video chunks."}
          </p>
          <Button
            variant="outline"
            size="sm"
            onClick={() => checkVideo(true)}
            className="mt-3 gap-1"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Retry Merge
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
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="gap-1 text-destructive hover:text-destructive"
              >
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
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="relative">
          <video
            src={videoUrl}
            controls
            className="w-full max-w-160 rounded-lg border bg-black"
          />
        </div>
        {error && (
          <div className="mt-2 flex items-center gap-1 text-sm text-destructive">
            <AlertCircle className="h-3 w-3" />
            {error}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

