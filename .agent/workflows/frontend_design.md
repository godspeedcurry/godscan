---
description: Enhance the HTML report design with modern, premium aesthetics
---
1. Analyze current report template
// turbo
view_code_item utils/report_html.go

2. Identify CSS improvements
   - Target the `const htmlTemplate` or equivalent variable.
   - Look for proper color palette (HSL), typography (Inter/Roboto), and layout (Flex/Grid).

3. Apply Design Principles
   - **Glassmorphism**: Add semi-transparent backgrounds with blur.
   - **Spacing**: Ensure ample white space (padding/margin).
   - **Interactive Elements**: Hover states for table rows and buttons.

4. Test changes
   - Re-run report generation: `go run main.go report`
   - Open generated HTML to verify.
