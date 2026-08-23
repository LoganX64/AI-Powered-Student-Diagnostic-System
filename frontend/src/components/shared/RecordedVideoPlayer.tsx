import { useEffect, useState, useCallback } from "react";
import { AlertCircle, Loader2, Film, Trash2 } from "lucide-react";
import { apiFetch } from "@/lib/api";
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
  has_video: boolean;
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  const checkVideo = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<VideoStatusResponse>(
        `/admin/assignments/${assignmentId}/video-chunks`
      );
      const exists = data.has_merged || (data.chunks && data.chunks.length > 0);
      setHasVideo(exists);

      if (exists) {
        const tokenData = await getVideoToken(assignmentId);
        setVideoToken(tokenData.token);
      }
    } catch (e) {
      setHasVideo(false);
    } finally {
      setLoading(false);
    }
  }, [assignmentId]);

  useEffect(() => {
    checkVideo();
  }, [checkVideo]);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await deleteVideo(assignmentId);
      setHasVideo(false);
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
    ? `${import.meta.env.VITE_BACKEND_URL}/admin/assignments/${assignmentId}/video-merged?token=${videoToken}`
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

  if (!hasVideo) {
    return null;
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
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="relative">
          <video
            src={videoUrl}
            controls
            className="w-full max-w-[640px] rounded-lg border bg-black"
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
