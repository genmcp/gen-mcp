// Add copy buttons to code blocks
document.addEventListener('DOMContentLoaded', function() {
  // Find all pre elements with code
  const codeBlocks = document.querySelectorAll('pre');

  codeBlocks.forEach((pre) => {
    // Create wrapper div
    const wrapper = document.createElement('div');
    wrapper.className = 'code-block-wrapper';

    // Wrap the pre element
    pre.parentNode.insertBefore(wrapper, pre);
    wrapper.appendChild(pre);

    // Create copy button
    const button = document.createElement('button');
    button.className = 'copy-code-button';
    button.setAttribute('aria-label', 'Copy code to clipboard');

    // Copy icon (two overlapping squares)
    const copyIcon = `
      <svg class="copy-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor">
        <rect x="5" y="5" width="9" height="9" rx="1" stroke-width="1.5"/>
        <rect x="2" y="2" width="9" height="9" rx="1" stroke-width="1.5"/>
      </svg>
    `;

    // Check icon
    const checkIcon = `
      <svg class="check-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor">
        <path d="M13.5 4L6 11.5L2.5 8" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    `;

    button.innerHTML = copyIcon + checkIcon;

    // Add click event
    button.addEventListener('click', async function() {
      const code = pre.querySelector('code');
      const text = code ? code.textContent : pre.textContent;

      try {
        await navigator.clipboard.writeText(text);

        // Show success state
        button.classList.add('copied');

        // Reset after 2 seconds
        setTimeout(() => {
          button.classList.remove('copied');
        }, 2000);
      } catch (err) {
        console.error('Failed to copy code:', err);

        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        document.body.appendChild(textArea);
        textArea.select();

        try {
          document.execCommand('copy');
          button.classList.add('copied');
          setTimeout(() => {
            button.classList.remove('copied');
          }, 2000);
        } catch (err) {
          console.error('Fallback copy failed:', err);
        }

        document.body.removeChild(textArea);
      }
    });

    wrapper.appendChild(button);
  });
});
