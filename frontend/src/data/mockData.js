export const mockDevice = {
  vendor: "TP-Link",
  model: "TL-WR841N/ND",
  hardwareVersion: {
    value: "WR841N v8 00000000",
    trust: "untrusted",
    source: "router:status"
  },
  firmwareVersion: {
    value: "3.15.9 Build 140724 Rel.63227n",
    trust: "untrusted",
    source: "router:status"
  },
  managementAddress: "192.168.1.1",
  authenticated: "unknown",
  provenance: "fixture"
};

export const mockStatus = {
  reachable: "true",
  wanStatus: "connected",
  uptimeSeconds: 20000,
  provenance: "fixture"
};

export const mockClients = {
  state: "verified",
  clients: [
    {
      name: "omarchy-laptop",
      mac: "00:11:22:33:44:55",
      ip: "192.168.1.100",
      lease: "01:24:35"
    },
    {
      name: "pixel-phone",
      mac: "AA:BB:CC:11:22:33",
      ip: "192.168.1.101",
      lease: "01:58:12"
    },
    {
      name: "smart-tv-livingroom",
      mac: "10:20:30:40:50:60",
      ip: "192.168.1.105",
      lease: "Permanent"
    }
  ]
};

export const mockCapabilities = {
  capabilities: {
    device: "verified",
    status: "verified",
    clients: "verified",
    wireless_security: "unsupported_or_unverified",
    wps: "unsupported_or_unverified",
    dmz: "unsupported_or_unverified",
    upnp: "unsupported_or_unverified",
    remote_management: "unsupported_or_unverified",
    forwarding: "unsupported_or_unverified"
  }
};

const fixtureObservation = (data) => ({
  state: "verified",
  reason: "Ejemplo sintético de fixture.",
  provenance: "fixture",
  data
});

const unverifiedSecurity = (reason) => ({
  state: "unsupported_or_unverified",
  reason,
  provenance: "absent",
  data: null
});

const fixtureText = (value) => ({ value, trust: "untrusted", source: "router:dhcp" });

export const mockObservations = {
  "/v0/device": fixtureObservation(mockDevice),
  "/v0/status": fixtureObservation(mockStatus),
  "/v0/clients": fixtureObservation(mockClients.clients.map(({ name, lease, ...client }) => ({
    ...client, name: fixtureText(name), leaseTime: fixtureText(lease), provenance: "fixture"
  }))),
  "/v0/capabilities": fixtureObservation(mockCapabilities.capabilities),
  "/v0/security/wireless": unverifiedSecurity("Solo hay evidencia histórica para el firmware 3.15.9; el fixture no aporta una lectura Wi-Fi."),
  "/v0/security/wps": unverifiedSecurity("El fixture no aporta una lectura verificada de WPS."),
  "/v0/security/dmz": unverifiedSecurity("El fixture no aporta una lectura verificada de DMZ.")
};
