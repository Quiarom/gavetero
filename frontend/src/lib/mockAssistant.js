import { mockObservations } from "../data/mockData.js";

function step(tool, path) {
  const result = mockObservations[path];
  return { tool, path, http_status: result.state === "unsupported_or_unverified" ? 404 : 200, result };
}

export function mockAssistantReply(question) {
  const query = question.toLowerCase();
  const intro = "Demostración con fixtures sintéticos; no es evidencia del router actual.";
  if (/wi-?fi|inalámbric|expuest|segur|wps|dmz|riesgo/.test(query)) {
    const labels = { wireless: "Wi-Fi", wps: "WPS", dmz: "DMZ" };
    const steps = Object.keys(labels).map((name) => step("get_security", `/v0/security/${name}`));
    const observations = steps.map(({ path, result }) =>
      `• ${labels[path.split("/").at(-1)]} (${result.state}): ${result.reason}`
    );
    return {
      steps,
      answer: [intro, ...observations, "Estos resultados no permiten determinar la configuración ni descartar riesgos."].join("\n\n")
    };
  }

  if (/qui[eé]n|dispositiv|conectad|aparato|dhcp/.test(query)) {
    const clients = step("get_clients", "/v0/clients");
    return {
      steps: [clients],
      answer: [intro, ...clients.result.data.map(({ name, ip, mac, leaseTime }) =>
        `• ${name.value} (IP: ${ip}, MAC: ${mac}, concesión: ${leaseTime.value})`),
      "Las concesiones DHCP no demuestran qué dispositivos están conectados ahora."].join("\n\n")
    };
  }

  const device = step("get_device", "/v0/device");
  const status = step("get_status", "/v0/status");
  const clients = step("get_clients", "/v0/clients");
  const capabilities = step("get_capabilities", "/v0/capabilities");
  const { vendor, model, firmwareVersion } = device.result.data;
  return {
    steps: [device, status, clients, capabilities],
    answer: [intro,
      `• Dispositivo de ejemplo: ${vendor} ${model} (firmware ${firmwareVersion.value}).`,
      `• Estado del enlace WAN de ejemplo: ${status.result.data.wanStatus}.`,
      `• Concesiones DHCP en el fixture: ${clients.result.data.length}; no son un censo.`
    ].join("\n\n")
  };
}
