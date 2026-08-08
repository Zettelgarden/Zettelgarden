/**
 * Processes template variables in text and replaces them with their actual values
 *
 * Supported variables:
 * - $date - Current date (YYYY-MM-DD)
 * - $time - Current time (HH:MM)
 * - $datetime - Full date and time (YYYY-MM-DD HH:MM)
 * - $day - Current day (numeric)
 * - $month - Current month (numeric)
 * - $year - Current year
 * - $weekday - Current day of the week (Monday, Tuesday, etc.)
 *
 * @param text The text containing template variables
 * @returns The text with variables replaced with their values
 */
export const processTemplateVariables = (text: string): string => {
  if (!text) return text;

  // Get current date/time information
  const now = new Date();

  // Define variable replacements
  const replacements: Record<string, string> = {
    $date: now.toISOString().split('T')[0], // YYYY-MM-DD
    $time: now.toTimeString().slice(0, 5), // HH:MM
    $datetime: now.toISOString().replace('T', ' ').slice(0, 16), // YYYY-MM-DD HH:MM
    $day: now.getDate().toString(),
    $month: (now.getMonth() + 1).toString(),
    $year: now.getFullYear().toString(),
    $weekday: now.toLocaleDateString('en-US', { weekday: 'long' }),
  };

  // Replace all variables
  let result = text;
  for (const [variable, value] of Object.entries(replacements)) {
    // Escape the $ character in the variable name for the regex
    const escapedVariable = variable.replace('$', '\\$');
    result = result.replace(new RegExp(escapedVariable, 'g'), value);
  }

  return result;
};
