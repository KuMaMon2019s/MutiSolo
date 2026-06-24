import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";

import type { Block, PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import { useCallback, useMemo, useState } from "react";

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
  objectUrl: string;
};

type ExportedAttachment = Omit<Attachment, "objectUrl"> & {
  source: "local_browser_attachment";
};

type EditorContext = {
  schemaVersion: 1;
  source: "mutisolo-requirement-editor";
  blocks: Block[];
  plainText: string;
  tencentDocs: TencentDoc[];
  attachments: ExportedAttachment[];
};

const defaultDraftKey = "mutisolo.requirementEditor.draft.v1";

function draftKeyFromSearch(search: string) {
  const params = new URLSearchParams(search);
  const project = params.get("project") || "local";
  const requirement = params.get("requirement") || "draft";
  return `mutisolo.requirementEditor.${project}.${requirement}.v1`;
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

function makeAttachment(file: File, objectUrl: string): Attachment {
  return {
    id: crypto.randomUUID(),
    name: file.name,
    mimeType: file.type || "application/octet-stream",
    size: file.size,
    kind: file.type.startsWith("image/") ? "image" : "file",
    objectUrl
  };
}

function sanitizeAttachments(attachments: Attachment[]): ExportedAttachment[] {
  return attachments.map(({ objectUrl: _objectUrl, ...attachment }) => ({
    ...attachment,
    source: "local_browser_attachment"
  }));
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
    const objectUrl = URL.createObjectURL(file);
    const attachment = makeAttachment(file, objectUrl);
    setAttachments((current) => [...current, attachment]);
    return {
      url: objectUrl,
      name: file.name
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
      source: "mutisolo-requirement-editor",
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

  return (
    <main className={`editorShell ${isEmbedded ? "embedded" : ""}`}>
      {!isEmbedded && (
        <header className="editorTopbar">
          <div>
            <p className="eyebrow">MutiSolo Requirement</p>
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
            }}
          />
        </section>
      </section>
    </main>
  );
}
