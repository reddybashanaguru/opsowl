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
      className="fixed z-50 bg-[#1e293b] border border-slate-600 rounded-lg shadow-xl py-1 min-w-[180px]"
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

export function NoteEditor({ caseId, initialContent = "", initialTitle = "", onSave, onCancel, isLoading, aiGenerated }: Props) {
  const [title, setTitle] = useState(initialTitle);
  const [noteType, setNoteType] = useState<"internal" | "formal">("internal");
  const [mentionedIds, setMentionedIds] = useState<Set<string>>(new Set());
  const [members, setMembers] = useState<CaseMember[]>([]);
  const [mentionItems, setMentionItems] = useState<CaseMember[]>([]);
  const [mentionRect, setMentionRect] = useState<DOMRect | null>(null);
  const mentionCommandRef = useRef<((item: CaseMember) => void) | null>(null);

  useEffect(() => {
    api.get<{ data: CaseMember[] }>(`/api/v1/users?limit=200`).then((res) => {
      const users = (res as unknown as { data: { users: CaseMember[] } }).data?.users ?? [];
      setMembers(users);
    }).catch(() => {});
  }, [caseId]);

  const editor = useEditor({
    extensions: [
      StarterKit,
      Placeholder.configure({ placeholder: "Write your findings, observations, or analysis here…" }),
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
  });

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
    <div className="bg-[#1e293b] border border-slate-600 rounded-xl overflow-hidden">
      {aiGenerated && (
        <div className="px-4 py-2 bg-violet-900/20 border-b border-violet-500/30 flex items-center gap-2">
          <span className="text-violet-400 text-xs font-semibold">✨ AI Generated</span>
          <span className="text-slate-500 text-xs">— Review and edit before saving</span>
        </div>
      )}

      {/* Title */}
      <div className="px-4 pt-4 pb-2">
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Note title (optional)"
          className="w-full bg-transparent text-white font-semibold text-base placeholder:text-slate-600 focus:outline-none border-b border-transparent focus:border-slate-600 pb-1 transition-colors"
        />
      </div>

      {/* Toolbar */}
      <div className="px-4 py-1.5 border-y border-slate-700/60 flex items-center gap-1">
        {[
          { cmd: () => editor?.chain().focus().toggleBold().run(), label: "B", title: "Bold", cls: "font-bold" },
          { cmd: () => editor?.chain().focus().toggleItalic().run(), label: "I", title: "Italic", cls: "italic" },
          { cmd: () => editor?.chain().focus().toggleStrike().run(), label: "S", title: "Strike", cls: "line-through" },
          { cmd: () => editor?.chain().focus().toggleBulletList().run(), label: "•", title: "Bullet list", cls: "" },
          { cmd: () => editor?.chain().focus().toggleOrderedList().run(), label: "1.", title: "Ordered list", cls: "" },
          { cmd: () => editor?.chain().focus().toggleCode().run(), label: "<>", title: "Code", cls: "font-mono text-xs" },
          { cmd: () => editor?.chain().focus().toggleBlockquote().run(), label: "❝", title: "Quote", cls: "" },
        ].map(({ cmd, label, title: ttl, cls }) => (
          <button
            key={label}
            onClick={cmd}
            title={ttl}
            className={`w-7 h-7 text-slate-400 hover:text-white hover:bg-slate-700 rounded text-xs flex items-center justify-center transition-colors ${cls}`}
          >
            {label}
          </button>
        ))}
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
            className="bg-slate-800 border border-slate-600 text-slate-300 text-xs rounded px-2 py-1 focus:outline-none focus:border-cyan-500"
          >
            <option value="internal">Internal</option>
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
