// tweaks.jsx — 1Trade deck tweaks
// Brand name, accent color, highlight style.

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "brandName": "1Trade",
  "contactEmail": "contact@1trade.com",
  "highlightStyle": "rough",
  "accentColor": "#C8F25C",
  "displayFont": "Inter Tight"
}/*EDITMODE-END*/;

const FONT_OPTIONS = [
  { name: "Inter Tight", stack: "'Inter Tight', 'Inter', sans-serif" },
  { name: "Inter",       stack: "'Inter', sans-serif" },
  { name: "Space Grotesk", stack: "'Space Grotesk', sans-serif" },
];

function loadGoogleFont(family) {
  const id = "gf-" + family.replace(/\W+/g, "-");
  if (document.getElementById(id)) return;
  const link = document.createElement("link");
  link.id = id;
  link.rel = "stylesheet";
  link.href = "https://fonts.googleapis.com/css2?family=" +
    encodeURIComponent(family) + ":wght@400;500;600;700&display=swap";
  document.head.appendChild(link);
}

function Trade1Tweaks() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);

  // Apply brand name + contact across DOM
  React.useEffect(() => {
    const upper = t.brandName.toUpperCase();
    document.querySelectorAll(".brand-name").forEach((el) => {
      // Use uppercase if styled in brand-line, otherwise as-is.
      const inBrandLine = el.closest(".brand-line");
      el.textContent = inBrandLine ? upper : t.brandName;
    });
    document.querySelectorAll(".contact-email").forEach((el) => {
      el.textContent = t.contactEmail;
    });
    document.title = t.brandName + " — Pitch Deck";
  }, [t.brandName, t.contactEmail]);

  // Highlight style on the deck-stage
  React.useEffect(() => {
    const stage = document.querySelector("deck-stage");
    if (stage) stage.setAttribute("data-hl", t.highlightStyle);
  }, [t.highlightStyle]);

  // Accent color
  React.useEffect(() => {
    document.documentElement.style.setProperty("--lime", t.accentColor);
  }, [t.accentColor]);

  // Display font
  React.useEffect(() => {
    const opt = FONT_OPTIONS.find((o) => o.name === t.displayFont) || FONT_OPTIONS[0];
    loadGoogleFont(opt.name);
    document.body.style.fontFamily = opt.stack;
  }, [t.displayFont]);

  return (
    <TweaksPanel title="Tweaks">
      <TweakSection label="Brand" />
      <TweakText
        label="Brand name"
        value={t.brandName}
        onChange={(v) => setTweak("brandName", v)}
      />
      <TweakText
        label="Contact email"
        value={t.contactEmail}
        onChange={(v) => setTweak("contactEmail", v)}
      />

      <TweakSection label="Typography" />
      <TweakSelect
        label="Display font"
        value={t.displayFont}
        options={FONT_OPTIONS.map((o) => o.name)}
        onChange={(v) => setTweak("displayFont", v)}
      />

      <TweakSection label="Highlight" />
      <TweakRadio
        label="Marker style"
        value={t.highlightStyle}
        options={["rough", "clean", "underline"]}
        onChange={(v) => setTweak("highlightStyle", v)}
      />
      <TweakColor
        label="Accent color"
        value={t.accentColor}
        options={["#C8F25C", "#D9FF4A", "#A8E635", "#FFCE22", "#F5A623", "#FF6B35"]}
        onChange={(v) => setTweak("accentColor", v)}
      />
    </TweaksPanel>
  );
}

const __twkRoot = ReactDOM.createRoot(document.getElementById("tweaks-root"));
__twkRoot.render(<Trade1Tweaks />);
