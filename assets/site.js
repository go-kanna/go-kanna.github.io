// Copy buttons, the demo video, and the workbench of the landing page. The landing page has a
// copy button beside each install line; a tool page gets one on every code
// block of its README. Without this script, each tab of the workbench is a link
// to its panel, and every panel shows.
(function () {
  const label = document.documentElement.lang === "ja"
    ? { copy: "コピー", copied: "コピーしました", select: "選択してコピーしてください" }
    : { copy: "Copy", copied: "Copied", select: "Select and copy" };

  function copyButton(text, button) {
    button.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(text());
        button.textContent = label.copied;
      } catch {
        button.textContent = label.select;
      }
      setTimeout(() => { button.textContent = label.copy; }, 2000);
    });
  }

  for (const button of document.querySelectorAll("[data-copy]")) {
    copyButton(() => document.getElementById(button.dataset.copy).textContent, button);
  }

  for (const block of document.querySelectorAll(".readme .highlight")) {
    const pre = block.querySelector("pre");
    if (!pre) continue;
    const button = document.createElement("button");
    button.type = "button";
    button.className = "copy";
    button.textContent = label.copy;
    copyButton(() => pre.textContent, button);
    block.append(button);
  }

  for (const link of document.querySelectorAll(".demo-watch[data-video]")) {
    link.addEventListener("click", (event) => {
      if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      event.preventDefault();
      const player = document.createElement("iframe");
      player.src = "https://www.youtube-nocookie.com/embed/" + link.dataset.video + "?autoplay=1";
      player.title = link.getAttribute("aria-label");
      player.allow = "autoplay; encrypted-media; fullscreen; picture-in-picture";
      player.referrerPolicy = "strict-origin-when-cross-origin";
      player.allowFullscreen = true;
      link.replaceWith(player);
      player.focus();
    });
  }

  // The workbench: one tab per tool, showing what you write and what it
  // writes.
  const tabs = [...document.querySelectorAll(".picker [role=tab]")];

  function select(tab, focus) {
    for (const t of tabs) {
      const on = t === tab;
      t.setAttribute("aria-selected", on);
      t.tabIndex = on ? 0 : -1;
      document.getElementById(t.getAttribute("aria-controls")).hidden = !on;
    }
    if (focus) tab.focus();
  }

  for (const [i, tab] of tabs.entries()) {
    tab.addEventListener("click", (e) => {
      e.preventDefault();
      select(tab, false);
    });
    tab.addEventListener("keydown", (e) => {
      const step = { ArrowDown: 1, ArrowRight: 1, ArrowUp: -1, ArrowLeft: -1 }[e.key];
      const to = step ? tabs[(i + step + tabs.length) % tabs.length]
        : e.key === "Home" ? tabs[0]
        : e.key === "End" ? tabs[tabs.length - 1]
        : null;
      if (!to) return;
      e.preventDefault();
      select(to, true);
    });
  }
  if (tabs.length) select(tabs.find((t) => t.getAttribute("aria-selected") === "true") || tabs[0], false);
})();
