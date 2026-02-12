/**
 * Spreadsheet API Client
 *
 * Provides functions for CRUD operations on spreadsheets attached to cards.
 * Spreadsheets are stored per-card and contain grid-based data with formula support.
 */

import { apiClient, getData } from './client';
import type { Spreadsheet, SpreadsheetData } from '../models/Spreadsheet';

/**
 * Process spreadsheet data from API
 * Converts date strings to Date objects for proper frontend handling
 */
function processSpreadsheetFromAPI(spreadsheet: any): Spreadsheet {
  return {
    ...spreadsheet,
    created_at: new Date(spreadsheet.created_at),
    updated_at: new Date(spreadsheet.updated_at),
  };
}

/**
 * Fetch all spreadsheets for a specific card
 *
 * @param cardId - The ID of the card to fetch spreadsheets for
 * @returns Promise<Spreadsheet[]> - Array of spreadsheets ordered by creation date
 *
 * @throws {NotFoundError} - If the card doesn't exist
 * @throws {AuthError} - If the user is not authenticated
 *
 * @example
 * ```ts
 * const spreadsheets = await fetchSpreadsheets(123);
 * console.log(`Found ${spreadsheets.length} spreadsheets`);
 * ```
 */
export async function fetchSpreadsheets(cardId: number): Promise<Spreadsheet[]> {
  const { data: spreadsheets } = await apiClient.get<Spreadsheet[]>(
    `/cards/${cardId}/spreadsheets`
  );

  if (!spreadsheets) {
    return [];
  }

  return spreadsheets.map(processSpreadsheetFromAPI);
}

/**
 * Fetch a single spreadsheet by ID
 *
 * @param id - The spreadsheet ID
 * @returns Promise<Spreadsheet> - The requested spreadsheet
 *
 * @throws {NotFoundError} - If the spreadsheet doesn't exist
 * @throws {AuthError} - If the user is not authenticated or doesn't own the spreadsheet
 *
 * @example
 * ```ts
 * const spreadsheet = await fetchSpreadsheet(1);
 * console.log(`Spreadsheet: ${spreadsheet.name}`);
 * ```
 */
export async function fetchSpreadsheet(id: number): Promise<Spreadsheet> {
  const { data: spreadsheet } = await apiClient.get<Spreadsheet>(
    `/spreadsheets/${id}`
  );
  return processSpreadsheetFromAPI(spreadsheet);
}

/**
 * Create a new spreadsheet attached to a card
 *
 * Creates a new 5x5 spreadsheet with the specified name. If no name is provided,
 * defaults to "sheet1".
 *
 * @param cardId - The ID of the card to attach the spreadsheet to
 * @param name - The name/identifier for the spreadsheet (default: "sheet1")
 * @returns Promise<Spreadsheet> - The newly created spreadsheet
 *
 * @throws {NotFoundError} - If the card doesn't exist
 * @throws {AuthError} - If the user is not authenticated or doesn't own the card
 *
 * @example
 * ```ts
 * // Create with default name
 * const spreadsheet1 = await createSpreadsheet(123, 'sheet1');
 *
 * // Create with custom name
 * const budgetSheet = await createSpreadsheet(123, 'budget');
 * ```
 */
export async function createSpreadsheet(
  cardId: number,
  name: string
): Promise<Spreadsheet> {
  const { data: spreadsheet } = await apiClient.post<Spreadsheet>(
    `/cards/${cardId}/spreadsheets`,
    { name }
  );
  return processSpreadsheetFromAPI(spreadsheet);
}

/**
 * Update an existing spreadsheet's data
 *
 * Updates the grid data, dimensions, and cell contents of a spreadsheet.
 * The entire data structure is replaced with the provided data.
 *
 * @param id - The spreadsheet ID
 * @param data - The new spreadsheet data (rows, cols, cell data)
 * @returns Promise<Spreadsheet> - The updated spreadsheet
 *
 * @throws {NotFoundError} - If the spreadsheet doesn't exist
 * @throws {AuthError} - If the user is not authenticated or doesn't own the spreadsheet
 *
 * @example
 * ```ts
 * const updatedSpreadsheet = await updateSpreadsheet(1, {
 *   rows: 10,
 *   cols: 10,
 *   data: {
 *     A1: { value: '100', formula: '' },
 *     B1: { value: '200', formula: '' },
 *     C1: { value: '300', formula: 'A1+B1', computed: 300 },
 *   },
 * });
 * ```
 */
export async function updateSpreadsheet(
  id: number,
  data: SpreadsheetData
): Promise<Spreadsheet> {
  const { data: spreadsheet } = await apiClient.put<Spreadsheet>(
    `/spreadsheets/${id}`,
    data
  );
  return processSpreadsheetFromAPI(spreadsheet);
}

/**
 * Delete a spreadsheet
 *
 * Permanently deletes a spreadsheet. This action cannot be undone.
 *
 * @param id - The spreadsheet ID
 * @returns Promise<void>
 *
 * @throws {NotFoundError} - If the spreadsheet doesn't exist
 * @throws {AuthError} - If the user is not authenticated or doesn't own the spreadsheet
 *
 * @example
 * ```ts
 * await deleteSpreadsheet(1);
 * console.log('Spreadsheet deleted');
 * ```
 */
export async function deleteSpreadsheet(id: number): Promise<void> {
  await apiClient.delete(`/spreadsheets/${id}`);
}
