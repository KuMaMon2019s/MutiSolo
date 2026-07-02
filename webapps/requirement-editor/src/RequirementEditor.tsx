import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";

import type { Block, PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import { useCallback, useEffect, useMemo, useState } from "react";

type TencentDoc = {
  type: "tencent_doc";
  title: string;
  url: string;
  readInstruction: string;
};

type Attachment = {
  id: string;
  name: string;
  mimeType: string;
  size: number;
  kind: "image" | "file";
  url: string;
  storageKey?: string;
  source: string;
};

type ExportedAttachment = Attachment;

type EditorContext = {
  schemaVersion: 1;
  source: "Mutesolo-requirement-editor";
  blocks: Block[];
  plainText: string;
  tencentDocs: TencentDoc[];
  attachments: ExportedAttachment[];
};

const defaultDraftKey = "Mutesolo.requirementEditor.draft.v1";

function draftKeyFromSearch(search: string) {
  const params = new URLSearchParams(search);
  const project = params.get("project") || "local";
  const requirement = params.get("requirement") || "draft";
  return `Mutesolo.requirementEditor.${project}.${requirement}.v1`;
}

function starterBlocksFromSearch(search: string): PartialBlock[] {
  const params = new URLSearchParams(search);
  const title = params.get("title") || "需求标题";
  const description = params.get("description") || "在这里补充功能需求、接口要求、边界条件和验收标准。";
  return [
    {
      type: "heading",
      props: { level: 2 },
      content: title
    },
    {
      type: "paragraph",
      content: description
    }
  ];
}

function readDraft(draftKey: string): { blocks?: PartialBlock[]; tencentDocs?: TencentDoc[]; attachments?: Attachment[] } {
  const raw = window.localStorage.getItem(draftKey);
  if (!raw) return {};
  try {
    return JSON.parse(raw) as { blocks?: PartialBlock[]; tencentDocs?: TencentDoc[]; attachments?: Attachment[] };
  } catch {
    return {};
  }
}

function sanitizeAttachments(attachments: Attachment[]): ExportedAttachment[] {
  return attachments.filter((attachment) => Boolean(attachment.url));
}

async function uploadAsset(file: File): Promise<Attachment> {
  const body = new FormData();
  body.append("file", file);
  const response = await fetch("/api/assets", {
    method: "POST",
    body
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || "Upload failed");
  }
  return {
    id: data.id || crypto.randomUUID(),
    name: data.name || file.name,
    mimeType: data.mimeType || file.type || "application/octet-stream",
    size: data.size || file.size,
    kind: data.kind === "image" ? "image" : "file",
    url: data.url,
    storageKey: data.storageKey,
    source: data.source || "minio"
  };
}

function textFromInlineContent(content: unknown): string {
  if (typeof content === "string") return content;
  if (!Array.isArray(content)) return "";
  return content
    .map((item) => {
      if (typeof item === "string") return item;
      if (!item || typeof item !== "object") return "";
      const candidate = item as { text?: unknown; content?: unknown };
      if (typeof candidate.text === "string") return candidate.text;
      return textFromInlineContent(candidate.content);
    })
    .join("");
}

function blockToText(block: Block): string {
  const parts = [textFromInlineContent(block.content)];
  if (Array.isArray(block.children)) {
    parts.push(...block.children.map((child) => blockToText(child as Block)));
  }
  return parts.filter(Boolean).join("\n");
}

function buildPlainText(blocks: Block[]): string {
  return blocks
    .map(blockToText)
    .map((part) => part.trim())
    .filter(Boolean)
    .join("\n\n");
}

function emptyTencentDoc(): TencentDoc {
  return {
    type: "tencent_doc",
    title: "",
    url: "",
    readInstruction: ""
  };
}

export function RequirementEditor() {
  const search = window.location.search;
  const isEmbedded = new URLSearchParams(search).get("embed") === "1";
  const draftKey = useMemo(() => draftKeyFromSearch(search || defaultDraftKey), [search]);
  const starterBlocks = useMemo(() => starterBlocksFromSearch(search), [search]);
  const draft = useMemo(() => readDraft(draftKey), [draftKey]);
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [tencentDocs, setTencentDocs] = useState<TencentDoc[]>(draft.tencentDocs?.length ? draft.tencentDocs : [emptyTencentDoc()]);
  const [attachments, setAttachments] = useState<Attachment[]>(draft.attachments || []);

  const uploadFile = useCallback(async (file: File) => {
    const attachment = await uploadAsset(file);
    setAttachments((current) => [...current, attachment]);
    return {
      props: {
        url: attachment.url,
        name: attachment.name
      }
    };
  }, []);

  const editor = useCreateBlockNote(
    {
      initialContent: draft.blocks?.length ? draft.blocks : starterBlocks,
      uploadFile
    },
    []
  );

  const buildContext = useCallback((): EditorContext => {
    const currentBlocks = editor.document as Block[];
    return {
      schemaVersion: 1,
      source: "Mutesolo-requirement-editor",
      blocks: currentBlocks,
      plainText: buildPlainText(currentBlocks),
      tencentDocs: tencentDocs.filter((doc) => doc.title.trim() || doc.url.trim() || doc.readInstruction.trim()),
      attachments: sanitizeAttachments(attachments)
    };
  }, [attachments, editor, tencentDocs]);

  const persistDraft = useCallback((currentBlocks: Block[]) => {
    window.localStorage.setItem(
      draftKey,
      JSON.stringify({
        blocks: currentBlocks,
        tencentDocs,
        attachments
      })
    );
    setBlocks(currentBlocks);
    const context = buildContext();
    window.localStorage.setItem(`${draftKey}.context`, JSON.stringify(context));
  }, [attachments, buildContext, draftKey, tencentDocs]);

  const saveDraft = () => {
    persistDraft(editor.document as Block[]);
  };

  const postHeight = useCallback(() => {
    if (!isEmbedded) return;
    const height = Math.max(
      document.documentElement.scrollHeight,
      document.documentElement.offsetHeight,
      document.documentElement.clientHeight,
      document.body.scrollHeight,
      document.body.offsetHeight,
      document.body.clientHeight
    );
    window.parent.postMessage(
      {
        type: "Mutesolo.requirementEditor.height",
        height: Math.ceil(height)
      },
      window.location.origin
    );
  }, [isEmbedded]);

  useEffect(() => {
    postHeight();
    const resizeObserver = new ResizeObserver(() => postHeight());
    resizeObserver.observe(document.documentElement);
    resizeObserver.observe(document.body);
    return () => resizeObserver.disconnect();
  }, [postHeight]);

  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      if (event.data?.type !== "Mutesolo.requirementEditor.requestContext") return;
      const context = buildContext();
      persistDraft(editor.document as Block[]);
      window.parent.postMessage(
        {
          type: "Mutesolo.requirementEditor.context",
          requestId: event.data.requestId,
          context
        },
        window.location.origin
      );
    };
    window.addEventListener("message", handleMessage);
    return () => window.removeEventListener("message", handleMessage);
  }, [buildContext, editor, persistDraft]);

  return (
    <main className={`editorShell ${isEmbedded ? "embedded" : ""}`}>
      {!isEmbedded && (
        <header className="editorTopbar">
          <div>
            <p className="eyebrow">Mutesolo Requirement</p>
            <h1>Requirement Detail</h1>
            <p>Use BlockNote to capture requirement content.</p>
          </div>
          <div className="actionRow">
            <button type="button" onClick={saveDraft}>
              保存草稿
            </button>
          </div>
        </header>
      )}

      <section className="editorGrid">
        <section className="editorPanel">
          <div className="panelHead">
            <div>
              <h2>需求正文</h2>
            </div>
            <span>{blocks.length || editor.document.length} blocks</span>
          </div>
          <BlockNoteView
            editor={editor}
            theme="dark"
            onChange={() => {
              persistDraft(editor.document as Block[]);
              requestAnimationFrame(postHeight);
            }}
          />
        </section>
      </section>
    </main>
  );
}
