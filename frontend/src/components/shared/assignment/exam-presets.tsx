import {
  ClipboardListIcon,
  ServerIcon,
  VideoIcon,
} from "lucide-react";
import type { IntegrityPolicy } from "@/services/dashboard.service";

/**
 * Exam presets shared by every surface that creates an assignment
 * (the Assign Test tab and the per-student Assign Test dialog), so the two
 * cannot drift into offering different integrity options.
 */

export const EMPTY_POLICY: IntegrityPolicy = {
  server_timing: false,
  autosave: false,
  video_proctoring: false,
  tab_switch_detect: false,
};

export type ExamPreset = "simple" | "backend" | "video";

export function presetPolicy(preset: ExamPreset): IntegrityPolicy {
  switch (preset) {
    case "backend":
      return { server_timing: true, autosave: true, video_proctoring: false, tab_switch_detect: true };
    case "video":
      return { server_timing: true, autosave: true, video_proctoring: true, tab_switch_detect: true };
    default:
      return { ...EMPTY_POLICY };
  }
}

export const PRESETS: { key: ExamPreset; title: string; icon: typeof ClipboardListIcon; desc: string }[] = [
  {
    key: "simple",
    title: "Simple",
    icon: ClipboardListIcon,
    desc: "Client-only timing. No server sync, autosave, or video.",
  },
  {
    key: "backend",
    title: "Backend sync",
    icon: ServerIcon,
    desc: "Server-authoritative timing + autosave + tab detection.",
  },
  {
    key: "video",
    title: "Video proctored",
    icon: VideoIcon,
    desc: "Everything in Backend sync, plus video recording.",
  },
];

/** Individual integrity toggles, shown beneath the preset cards. */
export const INTEGRITY_TOGGLES: {
  key: keyof IntegrityPolicy;
  label: string;
  desc: string;
}[] = [
  { key: "server_timing", label: "Server timing", desc: "Authoritative start/deadline" },
  { key: "autosave", label: "Autosave", desc: "Server-side answer backups" },
  { key: "tab_switch_detect", label: "Tab switch detection", desc: "Log visibility changes" },
  { key: "video_proctoring", label: "Video proctoring", desc: "Record-only chunks" },
];

export function samePolicy(a: IntegrityPolicy, b: IntegrityPolicy) {
  return (
    a.server_timing === b.server_timing &&
    a.autosave === b.autosave &&
    a.tab_switch_detect === b.tab_switch_detect &&
    a.video_proctoring === b.video_proctoring
  );
}