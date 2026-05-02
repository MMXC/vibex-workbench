<!-- SpecInteractiveMarkdown.svelte — Interactive markdown renderer for specs
     Renders spec content as markdown with clickable action buttons
     SLICE-spec-interactive-markdown-renderer
-->
<script lang="ts">
  interface Props {
    content?: string;
    onActionClick?: (actionId: string, specPath: string) => void;
  }

  let { content = '', onActionClick }: Props = $props();

  // Simple markdown rendering (headers, bold, code, links)
  function renderMarkdown(md: string): string {
    let html = md
      // Escape HTML
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      // Code blocks
      .replace(/```([\s\S]*?)```/g, '<pre class="md-code"><code>$1</code></pre>')
      // Inline code
      .replace(/`([^`]+)`/g, '<code class="md-inline-code">$1</code>')
      // Headers
      .replace(/^### (.+)$/gm, '<h3 class="md-h3">$1</h3>')
      .replace(/^## (.+)$/gm, '<h2 class="md-h2">$1</h2>')
      .replace(/^# (.+)$/gm, '<h1 class="md-h1">$1</h1>')
      // Bold
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      // Italic
      .replace(/\*(.+?)\*/g, '<em>$1</em>')
      // Lists
      .replace(/^- (.+)$/gm, '<li class="md-li">$1</li>')
      .replace(/^(\d+)\. (.+)$/gm, '<li class="md-li-num">$2</li>')
      // Links
      .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a class="md-link" href="$2" target="_blank" rel="noopener">$1</a>')
      // Paragraphs
      .replace(/\n\n/g, '</p><p class="md-p">')
      // Line breaks
      .replace(/\n/g, '<br/>');

    return `<p class="md-p">${html}</p>`;
  }

  function handleClick(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (target.tagName === 'A' || target.tagName === 'BUTTON') {
      e.preventDefault();
      // handle action links
    }
  }
</script>

<div class="md-renderer">
  <div class="md-content">
    {@html renderMarkdown(content)}
  </div>
</div>

<style>
  .md-renderer {
    height: 100%;
    overflow: auto;
    padding: 0.75rem;
    background: #0d0d0e;
    font-family: ui-sans-serif, system-ui, sans-serif;
    font-size: 13px;
    color: #e4e4e7;
    line-height: 1.6;
  }

  .md-content :global(.md-h1) {
    font-size: 18px;
    font-weight: 700;
    color: #fafafa;
    margin: 0 0 0.75rem;
    border-bottom: 1px solid #27272a;
    padding-bottom: 0.4rem;
  }

  .md-content :global(.md-h2) {
    font-size: 15px;
    font-weight: 600;
    color: #fafafa;
    margin: 1rem 0 0.5rem;
  }

  .md-content :global(.md-h3) {
    font-size: 13px;
    font-weight: 600;
    color: #e4e4e7;
    margin: 0.75rem 0 0.4rem;
  }

  .md-content :global(.md-p) {
    margin: 0 0 0.5rem;
  }

  .md-content :global(.md-code) {
    background: #1a1a1a;
    border: 1px solid #27272a;
    border-radius: 6px;
    padding: 0.6rem;
    overflow: auto;
    margin: 0.5rem 0;
  }

  .md-content :global(.md-code code) {
    font-family: ui-monospace, monospace;
    font-size: 11px;
    color: #d4d4d8;
    white-space: pre;
  }

  .md-content :global(.md-inline-code) {
    font-family: ui-monospace, monospace;
    font-size: 11px;
    background: #1a1a1a;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    color: #9fc0ff;
  }

  .md-content :global(.md-li) {
    margin-left: 1rem;
    list-style: disc;
  }

  .md-content :global(.md-li-num) {
    margin-left: 1rem;
    list-style: decimal;
  }

  .md-content :global(.md-link) {
    color: #7aa2ff;
    text-decoration: none;
  }

  .md-content :global(.md-link:hover) {
    text-decoration: underline;
  }

  .md-content :global(strong) {
    font-weight: 600;
    color: #fafafa;
  }

  .md-content :global(em) {
    font-style: italic;
    color: #a1a1aa;
  }
</style>