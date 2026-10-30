// Copy buttons, and the workbench of the landing page. The landing page has a
// copy button beside each install line; a tool page gets one on every code
// block of its README. Without this script, each tab of the workbench is a link
// to its panel, and every panel shows.
(function () {
  function copyButton(text, button) {
    button.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(text());
        button.textContent = "Copied";
      } catch {
        button.textContent = "Select and copy";
      }
      setTimeout(() => { button.textContent = "Copy"; }, 2000);
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
    button.textContent = "Copy";
    copyButton(() => pre.textContent, button);
    block.append(button);
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
