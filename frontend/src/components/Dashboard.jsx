import React, { useState, useEffect } from "react";
import { 
  AlertCircle, 
  RotateCw,
  Layers,
  ChevronDown,
  ChevronUp,
  Copy,
  Check
} from "lucide-react";
import { ConnectionQuality } from "@/components/ConnectionQuality";
import { HelpGuide } from "@/components/HelpGuide";
import { DeviceManager } from "@/components/DeviceManager";
import { CapabilitiesGrid } from "@/components/CapabilitiesGrid";
import {
  mockDevice,
  mockStatus,
  mockClients,
  mockCapabilities
} from "@/data/mockData";
import { getRouterData } from "@/lib/desktop";

const EMPTY_DEVICE = {
  vendor: "Router",
  model: "sin identificar",
  hardwareVersion: { value: "" },
  firmwareVersion: { value: "" },
  managementAddress: "",
  authenticated: "unknown"
};
const EMPTY_STATUS = {
  reachable: "unknown",
  wanStatus: "unknown",
  uptimeSeconds: null
};
const NO_CLIENTS = { state: "unavailable", clients: null };
const WAN_LABELS = {
  connected: "WAN CONECTADA",
  disconnected: "WAN DESCONECTADA",
  connecting: "WAN CONECTANDO",
  disabled: "WAN DESHABILITADA"
};

function formatUptime(seconds) {
  if (!Number.isFinite(seconds) || seconds < 0) return "NO DISPONIBLE";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${days ? `${days}d ` : ""}${hours}h ${minutes}m`;
}

function LiveDashboard({ onWanStatusChange }) {
  const [observations, setObservations] = useState({});
  const [fetchError, setFetchError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    let cancelled = false;
    const paths = ["/v0/device", "/v0/status", "/v0/clients", "/v0/capabilities"];
    Promise.all(paths.map(getRouterData))
      .then(([device, status, clients, capabilities]) => {
        if (cancelled) return;
        setObservations({ device, status, clients, capabilities });
        setFetchError(null);
        onWanStatusChange?.(verifiedData(status)?.wanStatus ?? "unknown");
      })
      .catch((error) => {
        if (cancelled) return;
        setObservations({});
        setFetchError(error instanceof Error ? error.message : "No se pudo consultar el servicio local");
        onWanStatusChange?.("unknown");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [onWanStatusChange, revision]);

  return <DashboardView isLive observations={observations} fetchError={fetchError} loading={loading} refreshData={() => {
    setLoading(true);
    setRevision(revision + 1);
  }} />;
}

export function Dashboard({ isLive, onWanStatusChange }) {
  return isLive ? <LiveDashboard onWanStatusChange={onWanStatusChange} /> : <DashboardView />;
}

function verifiedData(response) {
  return response?.status === 200 && response.data?.state === "verified"
    ? response.data.data
    : null;
}

function isIPAddress(value) {
  if (typeof value !== "string") return false;
  if (/^(?:0|[1-9]\d{0,2})(?:\.(?:0|[1-9]\d{0,2})){3}$/.test(value)) {
    return value.split(".").every((octet) => Number(octet) <= 255);
  }
  return value.includes(":") && /^[\da-f:.]+$/i.test(value) && URL.canParse(`http://[${value}]/`);
}

export function DashboardView({ isLive = false, observations = {}, fetchError, loading, refreshData }) {
  const [showAdvancedTelemetry, setShowAdvancedTelemetry] = useState(false);
  const [copiedIp, setCopiedIp] = useState(false);

  const identity = verifiedData(observations.device);
  const status = verifiedData(observations.status);
  const statusData = isLive ? status ?? EMPTY_STATUS : mockStatus;
  const hasIdentity = identity && (identity.authenticated === "true" || observations.device.data.provenance === "fixture");
  const deviceData = isLive ? (hasIdentity ? identity : EMPTY_DEVICE) : mockDevice;
  const leases = observations.clients?.data;
  const leaseRows = leases?.data;
  const hasLeaseRows = observations.clients?.status === 200 && Array.isArray(leaseRows)
    && ((leases.state === "verified" && leaseRows.length > 0) || (leases.state === "absent" && leaseRows.length === 0));
  const clientsData = !isLive ? mockClients : hasLeaseRows ? {
    state: leases.state,
    clients: leaseRows.map((client) => ({
      ...client,
      name: client.name?.value || (typeof client.name === "string" ? client.name : null)
    }))
  } : { ...NO_CLIENTS, state: leases?.state === "unsupported_or_unverified" ? leases.state : "unavailable" };
  const capabilities = isLive ? verifiedData(observations.capabilities) ?? {} : mockCapabilities.capabilities;
  const fixtureSource = isLive && Object.values(observations).some((response) => response.data?.provenance === "fixture");
  const errors = [];
  if (isLive && !loading) {
    if (fetchError) {
      errors.push(fetchError);
    } else {
      if (!hasIdentity) {
        errors.push(identity ? "No se pudo confirmar la autenticación del router" : observations.device?.data?.reason || "No se pudo identificar el router");
      }
      if (!status) errors.push(observations.status?.data?.reason || "El estado WAN no está disponible");
      if (!hasLeaseRows) errors.push(leases?.reason || "Las concesiones DHCP no están disponibles");
    }
  }

  const managementAddress = deviceData.managementAddress;
  const hasManagementIP = isIPAddress(managementAddress);
  const handleCopyIp = () => {
    navigator.clipboard.writeText(managementAddress);
    setCopiedIp(true);
    setTimeout(() => setCopiedIp(false), 2000);
  };

  const wanLabel = WAN_LABELS[statusData.wanStatus] ?? "WAN NO DISPONIBLE";
  const wanTone = statusData.wanStatus === "connected"
    ? ["bg-emerald-500", "text-emerald-400"]
    : statusData.wanStatus === "disconnected" || statusData.wanStatus === "disabled"
      ? ["bg-rose-500", "text-rose-400"]
      : ["bg-neutral-500", "text-neutral-300"];
  const leaseSummary = clientsData.clients
    ? `${clientsData.clients.length} CONCESIONES DHCP`
    : clientsData.state === "unsupported_or_unverified" ? "POR VERIFICAR" : "NO DISPONIBLE";

  return (
    <div className="w-full px-4 sm:px-6 lg:px-8 py-5 space-y-6 bg-black text-white font-sans">
      {/* Live Error Notification */}
      {errors.length > 0 && (
        <div role="alert" className="border-2 border-neutral-700 bg-neutral-900 p-3 text-xs text-rose-300 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 font-mono">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-rose-400 shrink-0" />
            <span>{[...new Set(errors)].join(" · ")}</span>
          </div>
          <span className="bg-black px-2 py-0.5 text-rose-300 border border-neutral-700 text-xs">
            Observación parcial o no disponible
          </span>
        </div>
      )}

      {/* ========================================================================= */}
      {/* 1. SECCIÓN PRIMORDIAL: DATOS DEL ROUTER, IP Y FIRMWARE (BRUTALISTA ELEVADO) */}
      {/* ========================================================================= */}
      <section className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-4 text-white">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-5 pt-1">
          <div className="space-y-1.5">
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="text-2xl sm:text-3xl lg:text-4xl font-black uppercase tracking-tight text-white font-sans">
                {deviceData.vendor} <span className="text-primary">{deviceData.model}</span>
              </h1>
              <div className="flex items-center gap-1.5 px-2.5 py-0.5 border border-neutral-800 bg-neutral-900 font-mono text-xs">
                <span className={`h-2 w-2 rounded-full ${wanTone[0]}`} />
                <span className={`font-bold uppercase ${wanTone[1]}`}>
                  {wanLabel}
                </span>
              </div>
            </div>
            {fixtureSource && <p className="text-xs text-neutral-400">Incluye datos de fixture, no evidencia de hardware.</p>}
          </div>

          {isLive && <button
            onClick={refreshData}
            disabled={loading}
            className="px-4 py-2 bg-neutral-900 hover:bg-neutral-800 border-2 border-neutral-700 text-xs font-mono font-bold uppercase tracking-wider text-white flex items-center gap-2 transition-colors cursor-pointer self-start sm:self-auto disabled:opacity-50"
          >
            <RotateCw className={`h-3.5 w-3.5 ${loading ? "animate-spin text-primary" : ""}`} />
            <span>{loading ? "Actualizando..." : "Actualizar"}</span>
          </button>}
        </div>

        {/* Minimal Horizontal Telemetry Ribbon (Brutalist style with border-l-2) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 pt-1 font-mono text-xs">
          {/* IP con 1-click copy */}
          <div className="border-l-2 border-primary pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">
              {managementAddress === "fixture" ? "FUENTE DE IDENTIDAD:" : "IP DE GESTIÓN:"}
            </span>
            {hasManagementIP ? (
              <button
                onClick={handleCopyIp}
                className="flex items-center gap-1.5 font-bold text-white hover:text-primary text-sm transition-colors cursor-pointer text-left w-full group"
                title="Clic para copiar dirección IP"
              >
                <span>{managementAddress}</span>
                {copiedIp ? (
                  <Check className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
                ) : (
                  <Copy className="h-3.5 w-3.5 text-neutral-500 group-hover:text-primary shrink-0" />
                )}
                <span className="text-[10px] text-neutral-500 group-hover:text-primary">
                  {copiedIp ? "¡COPIADA!" : "[COPIAR]"}
                </span>
              </button>
            ) : (
              <span className="font-bold text-white text-sm">{managementAddress === "fixture" ? "FIXTURE" : "NO DISPONIBLE"}</span>
            )}
          </div>

          {/* Dispositivos Activos */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">DHCP:</span>
            <span className="font-bold text-white text-sm block">
              {leaseSummary}
            </span>
          </div>

          {/* Seguridad de Red */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">SEGURIDAD:</span>
            <span className="font-bold text-neutral-400 text-sm block">
              CONSULTAR CAPACIDADES
            </span>
          </div>

          {/* Tiempo Encendido */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">TIEMPO ACTIVO:</span>
            <span className="font-bold text-white text-sm block truncate">
              {statusData.uptime || formatUptime(statusData.uptimeSeconds)}
            </span>
          </div>

          {/* Versión Firmware */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2 col-span-1 sm:col-span-2 lg:col-span-1">
            <span className="text-neutral-500 uppercase text-xs block font-medium">FIRMWARE:</span>
            <span className="font-bold text-white text-sm block truncate" title={deviceData.firmwareVersion?.value}>
              {deviceData.firmwareVersion?.value || "NO DISPONIBLE"}
            </span>
          </div>
        </div>
      </section>

      {/* 2. CALIDAD DE CONEXIÓN */}
      <section>
        <ConnectionQuality isLive={isLive} />
      </section>

      {/* 3. DISPOSITIVOS CONECTADOS */}
      <section>
        <DeviceManager
          clients={clientsData.clients}
          state={clientsData.state}
        />
      </section>

      {/* 4. GUÍA CLARA Y SIMPLE */}
      <section>
        <HelpGuide />
      </section>

      {/* 5. FUNCIONES DEL ROUTER (Colapsable) */}
      <section className="border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm text-white">
        <div 
          onClick={() => setShowAdvancedTelemetry(!showAdvancedTelemetry)}
          className="flex items-center justify-between cursor-pointer group"
        >
          <div className="flex items-center gap-2.5">
            <Layers className="h-4 w-4 text-primary" />
            <h4 className="text-sm sm:text-base font-bold font-mono uppercase text-white group-hover:text-primary transition-colors">
              Funciones del Router
            </h4>
          </div>

          <div className="p-1 border border-neutral-800 bg-neutral-900 group-hover:border-neutral-700 transition-colors">
            {showAdvancedTelemetry ? (
              <ChevronUp className="h-4 w-4 text-white" />
            ) : (
              <ChevronDown className="h-4 w-4 text-neutral-400" />
            )}
          </div>
        </div>

        {showAdvancedTelemetry && (
          <div className="mt-4 pt-4 border-t-2 border-neutral-800">
            <CapabilitiesGrid
              capabilities={capabilities}
              isLive={isLive}
            />
          </div>
        )}
      </section>
    </div>
  );
}

export default Dashboard;
