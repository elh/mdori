document.addEventListener("DOMContentLoaded", () => {
  const renderer = window.mdoriBeautifulMermaid;
  if (!renderer || !renderer.renderMermaidSVG) {
    return;
  }

  document.querySelectorAll(".mdori-mermaid").forEach((node) => {
    const code = node.querySelector("code");
    if (!code) {
      return;
    }

    try {
      const svg = renderer.renderMermaidSVG(code.textContent, {
        bg: "var(--color-bg)",
        fg: "var(--color-text)",
        line: "var(--color-border)",
        accent: "var(--color-link)",
        muted: "var(--color-muted)",
        transparent: true,
      });
      node.innerHTML = svg;
    } catch (error) {
      const message = document.createElement("p");
      message.className = "mdori-mermaid-error";
      message.textContent = error instanceof Error ? error.message : String(error);
      node.prepend(message);
    }
  });
});
