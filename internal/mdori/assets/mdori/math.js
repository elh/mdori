document.addEventListener("DOMContentLoaded", () => {
  if (!window.katex) {
    return;
  }

  document.querySelectorAll(".mdori-math").forEach((node) => {
    window.katex.render(node.textContent, node, {
      displayMode: node.classList.contains("mdori-math-display"),
      throwOnError: false,
      trust: false,
    });
  });
});
