---
layout: page
title: Documentation Site
---

# gen-mcp Documentation Site

This directory contains the gen-mcp documentation website, built with Jekyll and hosted on GitHub Pages.

## Local Development

To run the documentation site locally:

### Prerequisites

- Ruby 2.7 or higher
- Bundler

### Setup

```bash
cd docs
bundle install
```

### Run Locally

```bash
bundle exec jekyll serve
```

The site will be available at `http://localhost:4000/gen-mcp/`

### Watch for Changes

Jekyll automatically watches for file changes and rebuilds the site. Simply edit markdown files and refresh your browser.

## Site Structure

```
docs/
├── _config.yml           # Jekyll configuration
├── _layouts/             # HTML layouts
│   └── default.html      # Main layout template
├── _includes/            # Reusable components
│   ├── header.html       # Site header with navigation
│   └── footer.html       # Site footer
├── assets/               # Static assets
│   ├── css/              # Stylesheets
│   │   ├── main.css      # Main styles
│   │   └── syntax.css    # Code syntax highlighting
│   └── js/               # JavaScript
│       ├── theme-toggle.js    # Dark/light mode
│       └── navigation.js      # Smooth scrolling & nav
├── index.md              # Homepage
├── mcpfile.md            # MCP File documentation
└── mcpserver.md          # MCP Server Config File documentation
```

## Adding New Pages

1. Create a new `.md` file in the `docs/` directory
2. Add front matter at the top:

```yaml
---
layout: default
title: Your Page Title
description: Page description for SEO
---
```

3. Write your content in Markdown
4. Link to it from other pages using: `[Link Text]({{ '/your-page.html' | relative_url }})`

## Theme & Styling

### Color Scheme

The site uses a custom color scheme with light and dark mode support:

**Light Mode:**
- Background: `#F7F9FC` (cool-toned off-white)
- Text: `#1A1A1A` (off-black)
- Accent Orange: `#E6622A`
- Accent Blue: `#EBFDFE`

**Dark Mode:**
- Background: `#0D1117`
- Text: `#E6EDF3`
- Same accent colors

### Typography

- **Headers:** Funnel Display (Google Font)
- **Body:** Space Grotesk (Google Font)

### Modifying Styles

Edit `assets/css/main.css` to customize the appearance. The file uses CSS variables for easy theming.

## Deployment

The site is automatically deployed to GitHub Pages when changes are pushed to the `main` branch. GitHub Pages builds the Jekyll site automatically.

### GitHub Pages Configuration

Make sure the following is set in your repository settings:
- Source: Deploy from a branch
- Branch: `main`
- Folder: `/docs`

## Features

- ✅ Responsive design (mobile-friendly)
- ✅ Dark/light mode toggle with localStorage persistence
- ✅ Smooth scrolling navigation
- ✅ Syntax highlighting for code blocks
- ✅ SEO-friendly
- ✅ Fast loading times

## Contributing

When contributing documentation:

1. Keep content in Markdown format
2. Use semantic HTML in layouts
3. Follow the existing color scheme
4. Test locally before submitting
5. Ensure mobile responsiveness

## License

Apache 2.0 - See LICENSE in the repository root.
