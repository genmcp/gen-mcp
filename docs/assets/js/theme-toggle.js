// Theme toggle functionality
(function() {
  const THEME_KEY = 'genmcp-theme';
  const themeToggle = document.getElementById('theme-toggle');

  // Get saved theme or default to light
  function getSavedTheme() {
    return localStorage.getItem(THEME_KEY) || 'light';
  }

  // Set theme
  function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(THEME_KEY, theme);
  }

  // Toggle theme
  function toggleTheme() {
    const currentTheme = getSavedTheme();
    const newTheme = currentTheme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
  }

  // Initialize theme on page load
  setTheme(getSavedTheme());

  // Add click event listener
  if (themeToggle) {
    themeToggle.addEventListener('click', toggleTheme);
  }

  // Listen for system theme preference changes
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)');
  prefersDark.addEventListener('change', (e) => {
    // Only auto-switch if user hasn't manually set a preference
    if (!localStorage.getItem(THEME_KEY)) {
      setTheme(e.matches ? 'dark' : 'light');
    }
  });
})();
