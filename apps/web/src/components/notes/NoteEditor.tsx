"use client";

import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Mention from "@tiptap/extension-mention";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { api } from "@/lib/api";

interface CaseMember {
  id: string;
  full_name: string;
  role: string;
}

interface Props {
  caseId: string;
  initialContent?: string;
  initialTitle?: string;
  onSave: (data: {
    title: string;
    content: string;
    contentText: string;
    noteType: "internal" | "formal";
    mentionedUserIds: string[];
  }) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
  aiGenerated?: boolean;
}

function MentionList({
  items,
  command,
  rect,
}: {
  items: CaseMember[];
  command: (item: CaseMember) => void;
  rect: DOMRect | null;
}) {
  if (!rect || items.length === 0) return null;
  return createPortal(
    <div
      className="fixed z-50 bg-[#131f30] border border-slate-600 rounded-lg shadow-xl py-1 min-w-[180px]"
      style={{ top: rect.bottom + 4, left: rect.left }}
    >
      {items.map((m) => (
        <button
          key={m.id}
          onClick={() => command(m)}
          className="w-full text-left px-3 py-2 text-sm text-slate-300 hover:bg-slate-700/60 flex items-center gap-2"
        >
          <span className="w-6 h-6 bg-cyan-700 rounded-full flex items-center justify-center text-[10px] font-bold text-white flex-shrink-0">
            {m.full_name.charAt(0).toUpperCase()}
          </span>
          <span>{m.full_name}</span>
          <span className="text-xs text-slate-500 capitalize">{m.role.replace("_", " ")}</span>
        </button>
      ))}
    </div>,
    document.body
  );
}

const NOTE_TYPE_CONFIG = {
  internal: {
    titlePlaceholder: "e.g. Customer call — 18 Jun 2026",
    bodyPlaceholder:  "e.g. Called Arjun Mehta. He claims the 9 deposits were advance payments to Nexgen Traders for raw materials. Unable to provide invoices on the spot — will courier within 3 days. Follow-up due 21 Jun.",
    borderColor:      "border-slate-600",
    badge:            { label: "Internal",          style: "bg-slate-700 text-slate-300" },
    hint:             "Only visible to your team. Use for working notes, call logs, and observations.",
  },
  formal: {
    titlePlaceholder: "e.g. MLRO Escalation Decision — Arjun Mehta Structuring Case",
    bodyPlaceholder:  "e.g. After reviewing transaction records, entity incorporation dates, and GST status, this case has been escalated to the MLRO for STR filing decision. Structuring pattern across 9 deposits confirmed. Three receiving entities (Nexgen Traders, Skyline Ventures, Primo Infra) incorporated within 6 months and have no GST registration. Risk level: High. Recommended action: File STR with FIU-IND.",
    borderColor:      "border-amber-500/60",
    badge:            { label: "Formal · Exportable", style: "bg-amber-900/30 text-amber-300 border border-amber-500/40" },
    hint:             "Part of the official record. Included in PDF exports and visible to auditors.",
  },
} as const;

export function NoteEditor({ caseId, initialContent = "", initialTitle = "", onSave, onCancel, isLoading, aiGenerated }: Props) {
  const [title, setTitle] = useState(initialTitle);
  const [noteType, setNoteType] = useState<"internal" | "formal">("internal");
  const [mentionedIds, setMentionedIds] = useState<Set<string>>(new Set());
  const [members, setMembers] = useState<CaseMember[]>([]);
  const [mentionItems, setMentionItems] = useState<CaseMember[]>([]);
  const [mentionRect, setMentionRect] = useState<DOMRect | null>(null);
  const mentionCommandRef = useRef<((item: CaseMember) => void) | null>(null);
  const typeConfig = NOTE_TYPE_CONFIG[noteType];

  useEffect(() => {
    api.get<{ data: CaseMember[] }>(`/api/v1/users?limit=200`).then((res) => {
      const users = (res as unknown as { data: { users: CaseMember[] } }).data?.users ?? [];
      setMembers(users);
    }).catch(() => {});
  }, [caseId]);

  const editor = useEditor({
    extensions: [
      StarterKit,
      Placeholder.configure({ placeholder: typeConfig.bodyPlaceholder }),
      Mention.configure({
        HTMLAttributes: { class: "mention" },
        suggestion: {
          items: ({ query }) => {
            return members.filter((m) =>
              m.full_name.toLowerCase().includes(query.toLowerCase())
            ).slice(0, 5);
          },
          render: () => {
            let rect: DOMRect | null = null;
            return {
              onStart: (props) => {
                setMentionItems(props.items as CaseMember[]);
                const domRect = props.clientRect?.();
                rect = domRect ?? null;
                setMentionRect(rect);
                mentionCommandRef.current = (item: CaseMember) => {
                  props.command({ id: item.id, label: item.full_name });
                  setMentionedIds((prev) => new Set([...prev, item.id]));
                  setMentionItems([]);
                  setMentionRect(null);
                };
              },
              onUpdate: (props) => {
                setMentionItems(props.items as CaseMember[]);
                const domRect = props.clientRect?.();
                rect = domRect ?? null;
                setMentionRect(rect);
              },
              onExit: () => {
                setMentionItems([]);
                setMentionRect(null);
              },
            };
          },
        },
      }),
    ],
    content: initialContent,
    editorProps: {
      attributes: {
        class: "prose prose-invert prose-sm max-w-none focus:outline-none min-h-[120px] text-slate-300 leading-relaxed",
      },
    },
  }, [noteType]);

  const handleSave = async () => {
    if (!editor) return;
    const html = editor.getHTML();
    const text = editor.getText();
    await onSave({
      title,
      content: html,
      contentText: text,
      noteType,
      mentionedUserIds: Array.from(mentionedIds),
    });
  };

  return (
    <div className={`bg-[#131f30] border rounded-xl overflow-hidden transition-colors ${typeConfig.borderColor}`}>
      {/* Type indicator banner */}
      <div className={`px-4 py-2 border-b flex items-center justify-between ${
        noteType === "formal"
          ? "bg-amber-900/10 border-amber-500/20"
          : "bg-slate-800/40 border-slate-700/60"
      }`}>
        <div className="flex items-center gap-2">
          <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${typeConfig.badge.style}`}>
            {typeConfig.badge.label}
          </span>
          <span className="text-slate-500 text-xs">{typeConfig.hint}</span>
        </div>
        {aiGenerated && (
          <span className="text-violet-400 text-xs font-semibold flex items-center gap-1">
            ✨ AI Generated — review before saving
          </span>
        )}
      </div>

      {/* Title */}
      <div className="px-4 pt-4 pb-2">
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder={typeConfig.titlePlaceholder}
          className="w-full bg-transparent text-white font-semibold text-base placeholder:text-slate-600 focus:outline-none border-b border-transparent focus:border-slate-600 pb-1 transition-colors"
        />
      </div>

      {/* Toolbar */}
      <div className="px-4 py-1.5 border-y border-slate-700/60 flex items-center gap-0.5">
        {[
          {
            cmd: () => editor?.chain().focus().toggleBold().run(),
            active: () => !!editor?.isActive("bold"),
            title: "Bold (Ctrl+B)",
            icon: <span className="font-bold text-xs">B</span>,
          },
          {
            cmd: () => editor?.chain().focus().toggleItalic().run(),
            active: () => !!editor?.isActive("italic"),
            title: "Italic (Ctrl+I)",
            icon: <span className="italic font-serif text-sm">I</span>,
          },
          {
            cmd: () => editor?.chain().focus().toggleStrike().run(),
            active: () => !!editor?.isActive("strike"),
            title: "Strikethrough",
            icon: <span className="line-through text-xs">S</span>,
          },
          { divider: true },
          {
            cmd: () => editor?.chain().focus().toggleBulletList().run(),
            active: () => !!editor?.isActive("bulletList"),
            title: "Bullet list — unordered list of items",
            icon: (
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
              </svg>
            ),
          },
          {
            cmd: () => editor?.chain().focus().toggleOrderedList().run(),
            active: () => !!editor?.isActive("orderedList"),
            title: "Numbered list — ordered list of items",
            icon: <span className="text-xs font-mono">1.</span>,
          },
          { divider: true },
          {
            cmd: () => editor?.chain().focus().toggleCode().run(),
            active: () => !!editor?.isActive("code"),
            title: "Inline code — highlight a value, ID, or code snippet",
            icon: <span className="font-mono text-xs">{`<>`}</span>,
          },
          {
            cmd: () => editor?.chain().focus().toggleBlockquote().run(),
            active: () => !!editor?.isActive("blockquote"),
            title: "Blockquote — highlight a quoted statement or important excerpt",
            icon: (
              <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                <path d="M6 17h3l2-4V7H5v6h3zm8 0h3l2-4V7h-6v6h3z" />
              </svg>
            ),
          },
        ].map((item, i) => {
          if ("divider" in item) {
            return <div key={i} className="w-px h-4 bg-slate-700 mx-1" />;
          }
          const isActive = item.active();
          return (
            <button
              key={i}
              onClick={item.cmd}
              title={item.title}
              className={`w-7 h-7 rounded text-xs flex items-center justify-center transition-colors ${
                isActive
                  ? "bg-slate-600 text-white"
                  : "text-slate-400 hover:text-white hover:bg-slate-700"
              }`}
            >
              {item.icon}
            </button>
          );
        })}
      </div>

      {/* Editor area */}
      <div className="px-4 py-3">
        <EditorContent editor={editor} />
      </div>

      {/* Mention dropdown */}
      <MentionList
        items={mentionItems}
        command={(item) => mentionCommandRef.current?.(item)}
        rect={mentionRect}
      />

      {/* Footer */}
      <div className="px-4 py-3 border-t border-slate-700/60 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <select
            value={noteType}
            onChange={(e) => setNoteType(e.target.value as "internal" | "formal")}
            className={`border text-xs rounded px-2 py-1 focus:outline-none transition-colors ${
              noteType === "formal"
                ? "bg-amber-900/20 border-amber-500/50 text-amber-300 focus:border-amber-400"
                : "bg-slate-800 border-slate-600 text-slate-300 focus:border-cyan-500"
            }`}
          >
            <option value="internal">Internal (team only)</option>
            <option value="formal">Formal (exportable)</option>
          </select>
          {mentionedIds.size > 0 && (
            <span className="text-xs text-violet-400">@{mentionedIds.size} mentioned</span>
          )}
        </div>
        <div className="flex gap-2">
          <button
            onClick={onCancel}
            className="text-slate-400 hover:text-slate-200 text-sm px-3 py-1.5 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isLoading}
            className="bg-cyan-500 hover:bg-cyan-400 text-slate-900 font-semibold text-sm px-4 py-1.5 rounded-lg transition-colors disabled:opacity-50"
          >
            {isLoading ? "Saving…" : "Save Note"}
          </button>
        </div>
      </div>
    </div>
  );
}
