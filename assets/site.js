// Copy buttons. The landing page has one beside each install line; a tool page
// gets one on every code block of its README.
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
})();
