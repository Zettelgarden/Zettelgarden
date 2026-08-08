export interface GenericResponse {
  message: string;
  error: boolean;
}

export interface ErrorResponse {
  error: string;
}

export function isErrorResponse(response: any): response is ErrorResponse {
  return response && typeof response.error === 'string';
}
