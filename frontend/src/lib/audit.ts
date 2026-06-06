import { getTransport } from "./api";

export type { AuditLogItem, AuditLogListResult } from "./api";

export async function listAuditLogs(filters: {
  q?: string;
  actorLabel?: string;
  entityType?: string;
  action?: string;
  dateFrom?: string;
  dateTo?: string;
  page?: number;
  pageSize?: number;
}) {
  const transport = await getTransport();
  return transport.listAuditLogs(filters);
}
