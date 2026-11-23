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
  authenticated: "true",
  provenance: "fixture"
};

export const mockStatus = {
  reachable: "true",
  wanStatus: "connected",
  uptimeSeconds: 20000,
  uptime: "5h33m20s",
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
