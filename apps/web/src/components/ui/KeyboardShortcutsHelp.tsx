"use client";

import { useEffect } from "react";
import { X, Keyboard } from "lucide-react";

interface ShortcutEntry {
  keys: string[];
  label: string;
}

interface ShortcutGroup {
  group: string;
  shortcuts: ShortcutEntry[];
}

const SHORTCUT_GROUPS: ShortcutGroup[] = [
  {
    group: "Navigation",
    shortcuts: [
      { keys: ["g", "d"], label: "Go to Dashboard" },
      { keys: ["g", "c"], label: "Go to Cases" },
      { keys: ["g", "a"], label: "Go to Alert Queue" },
      { keys: ["g", "s"], label: "Go to Search" },
      { keys: ["/"],      label: "Focus search" },
      { keys: ["?"],      label: "Show this help" },
    ],
  },
  {
    group: "Alert Queue",
    shortcuts: [
      { keys: ["j"],      label: "Move focus down" },
      { keys: ["k"],      label: "Move focus up" },
      { keys: ["x"],      label: "Select / deselect focused alert" },
      { keys: ["d"],      label: "Dismiss focused alert" },
      { keys: ["c"],      label: "Convert focused alert to case" },
      { keys: ["Esc"],    label: "Clear selection / close modal" },
    ],
  },
];

interface Props {
  onClose: () => void;
}

export function KeyboardShortcutsHelp({ onClose }: Props) {
  useEffect(() => {
    function handle(e: KeyboardEvent) {
      if (e.key === "Escape" || e.key === "?") onClose();
    }
    window.addEventListener("keydown", handle);
    return () => window.removeEventListener("keydown", handle);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-[#0f1c2e] border border-[#1e3050] rounded-2xl ring-1 ring-white/[0.04]
        w-full max-w-md p-6 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2.5">
            <Keyboard className="w-4 h-4 text-cyan-400" />
            <h2 className="text-slate-100 font-semibold text-base">Keyboard Shortcuts</h2>
          </div>
          <button onClick={onClose}
            className="p-1 rounded-lg text-slate-500 hover:text-slate-300 transition-colors">
            <X size={16} />
          </button>
        </div>

        {/* Groups */}
        <div className="space-y-5">
          {SHORTCUT_GROUPS.map(({ group, shortcuts }) => (
            <div key={group}>
              <div className="text-[10px] font-semibold text-slate-600 uppercase tracking-widest mb-2.5">
                {group}
              </div>
              <div className="space-y-1.5">
                {shortcuts.map(({ keys, label }) => (
                  <div key={label} className="flex items-center justify-between">
                    <span className="text-slate-400 text-sm">{label}</span>
                    <div className="flex items-center gap-1">
                      {keys.map((k, i) => (
                        <span key={i} className="flex items-center gap-1">
                          <kbd className="px-2 py-0.5 rounded-md bg-slate-800 border border-slate-700
                            text-slate-300 text-[11px] font-mono shadow-[0_1px_0_rgba(0,0,0,0.4)]">
                            {k}
                          </kbd>
                          {i < keys.length - 1 && (
                            <span className="text-slate-600 text-[10px]">then</span>
                          )}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <p className="text-slate-600 text-xs mt-5 text-center">
          Shortcuts are disabled while typing in inputs
        </p>
      </div>
    </div>
  );
}
