import React, { useState } from "react";
import {
  ArrowRight,
  Check,
  Copy,
  Cpu,
  Download,
  ExternalLink,
  Eye,
  Lock,
  Server,
  ShieldCheck,
  Sparkles,
  Terminal,
  Zap
} from "lucide-react";
import { Button } from "@/components/ui/button";

export function LandingPage({ onOpenDashboard }) {
  const [copiedCurl, setCopiedCurl] = useState(false);
  const curlCommand = "curl -sSf https://raw.githubusercontent.com/Quiarom/router-core/integration/gavetero/install.sh | sh";

  const copyCurl = () => {
    navigator.clipboard.writeText(curlCommand);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  return (
    <div className="mx-auto max-w-4xl px-4 sm:px-6 py-12 space-y-20 text-neutral-200 font-sans">
      {/* 1. HERO LIMPIO */}
      <header className="space-y-6 text-center sm:text-left pt-2">
        <div className="inline-flex items-center gap-2 border-2 border-primary bg-primary/10 px-3 py-1 text-xs font-mono font-bold text-primary uppercase">
          <Sparkles className="h-3.5 w-3.5" />
          <span>GMI Cloud × MiniMax Week 2026</span>
          <span className="text-neutral-500">•</span>
          <span className="text-white">Track: Reasoning</span>
        </div>

        <h1 className="text-3xl sm:text-5xl font-black uppercase tracking-tight text-white font-mono leading-tight">
          Control Plane Local con IA para Routers Legados
        </h1>

        <p className="text-base sm:text-lg text-neutral-300 leading-relaxed max-w-3xl">
          Convierte paneles de administración antiguos en una <strong className="text-white font-mono">API HTTP tipada en loopback</strong> y un <strong className="text-primary font-mono">agente de razonamiento MiniMax M3</strong> servido en <strong className="text-white">GMI Cloud</strong>. 100% solo lectura, cero riesgo de desconfiguración y privacidad total RFC1918.
        </p>

        {/* CTAs */}
        <div className="flex flex-wrap items-center justify-center sm:justify-start gap-3 pt-2 font-mono text-xs">
          <a
            href="#instalar"
            className="h-11 px-5 bg-primary hover:bg-primary-hover text-white font-bold uppercase tracking-wider flex items-center gap-2 border-2 border-primary transition-colors cursor-pointer"
          >
            <Download className="h-4 w-4" />
            <span>Instalar o Descargar</span>
          </a>

          <Button
            variant="outline"
            onClick={() => onOpenDashboard && onOpenDashboard()}
            className="h-11 px-5 gap-2 border-2 border-neutral-700 bg-neutral-900 text-neutral-200 hover:border-white hover:text-white"
          >
            <Zap className="h-4 w-4 text-emerald-400" />
            <span>Probar Asistente en Vivo</span>
            <ArrowRight className="h-4 w-4" />
          </Button>

          <a
            href="https://github.com/Quiarom/router-core"
            target="_blank"
            rel="noreferrer"
            className="h-11 px-4 bg-black hover:bg-neutral-900 border-2 border-neutral-800 text-neutral-400 hover:text-white flex items-center gap-2 transition-colors"
          >
            <ExternalLink className="h-4 w-4" />
            <span>GitHub</span>
          </a>
        </div>
      </header>

      {/* 2. INSTALACIÓN Y DESCARGAS (DIRECTO Y SIN CARDS ANIDADAS) */}
      <section id="instalar" className="border-t-2 border-neutral-800 pt-12 space-y-6">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono font-bold text-primary uppercase mb-1">
            <Terminal className="h-4 w-4" />
            <span>Instalación Rápida</span>
          </div>
          <h2 className="text-2xl font-black uppercase tracking-tight text-white font-mono">
            1 Comando en Terminal (CLI gavetero)
          </h2>
          <p className="mt-1 text-sm text-neutral-400">
            Descarga automática para Linux y macOS (AMD64 y ARM64), sin permisos de superusuario ni dependencias externas.
          </p>
        </div>

        {/* Terminal Box */}
        <div className="flex items-center justify-between gap-3 border-2 border-neutral-800 bg-black p-3.5 font-mono text-xs">
          <div className="flex items-center gap-2 overflow-x-auto text-neutral-200 pl-1">
            <span className="text-primary font-bold">$</span>
            <span className="select-all truncate">{curlCommand}</span>
          </div>
          <button
            onClick={copyCurl}
            className="flex items-center gap-1.5 shrink-0 border-2 border-neutral-700 bg-neutral-900 hover:border-primary hover:text-primary px-3 py-1.5 text-xs font-bold uppercase text-neutral-300 transition-colors cursor-pointer"
          >
            {copiedCurl ? (
              <>
                <Check className="h-3.5 w-3.5 text-emerald-400" />
                <span className="text-emerald-400">Copiado</span>
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                <span>Copiar</span>
              </>
            )}
          </button>
        </div>

        {/* Quick Commands List */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 font-mono text-xs pt-1">
          <div className="border border-neutral-800 bg-neutral-950 p-3 space-y-1">
            <div className="text-white font-bold">$ gvt setup</div>
            <div className="text-neutral-500 text-[11px]">Configura tu API Key de GMI Cloud.</div>
          </div>
          <div className="border border-neutral-800 bg-neutral-950 p-3 space-y-1">
            <div className="text-white font-bold">$ gvt inspect</div>
            <div className="text-neutral-500 text-[11px]">Inspecciona router y capacidades.</div>
          </div>
          <div className="border border-neutral-800 bg-neutral-950 p-3 space-y-1">
            <div className="text-white font-bold">$ gvt ask "pregunta"</div>
            <div className="text-neutral-500 text-[11px]">Razona y audita con MiniMax M3.</div>
          </div>
        </div>

        {/* Desktop App Download Buttons */}
        <div className="pt-4 space-y-3">
          <div className="text-xs font-mono font-bold uppercase text-neutral-400">
            O descarga la App de Escritorio con Interfaz Gráfica (Tauri 2):
          </div>
          <div className="flex flex-wrap gap-2.5 font-mono text-xs">
            <a
              href="https://github.com/Quiarom/router-core/releases/latest"
              target="_blank"
              rel="noreferrer"
              className="px-4 py-2.5 border-2 border-neutral-800 bg-neutral-950 hover:border-primary hover:text-white text-neutral-300 flex items-center gap-2 transition-colors"
            >
              <Download className="h-3.5 w-3.5 text-primary" />
              <span>Linux (.AppImage / .deb / .rpm)</span>
            </a>

            <a
              href="https://github.com/Quiarom/router-core/releases/latest"
              target="_blank"
              rel="noreferrer"
              className="px-4 py-2.5 border-2 border-neutral-800 bg-neutral-950 hover:border-primary hover:text-white text-neutral-300 flex items-center gap-2 transition-colors"
            >
              <Download className="h-3.5 w-3.5 text-primary" />
              <span>macOS (.dmg Apple Silicon / Intel)</span>
            </a>

            <a
              href="https://github.com/Quiarom/router-core/releases/latest"
              target="_blank"
              rel="noreferrer"
              className="px-4 py-2.5 border-2 border-neutral-800 bg-neutral-950 hover:border-primary hover:text-white text-neutral-300 flex items-center gap-2 transition-colors"
            >
              <Download className="h-3.5 w-3.5 text-primary" />
              <span>Windows (.exe / .zip)</span>
            </a>
          </div>
        </div>
      </section>

      {/* 3. SOBRE EL PROYECTO & SEGURIDAD */}
      <section className="border-t-2 border-neutral-800 pt-12 space-y-8">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono font-bold text-emerald-400 uppercase mb-1">
            <ShieldCheck className="h-4 w-4" />
            <span>Arquitectura y Filosofía</span>
          </div>
          <h2 className="text-2xl font-black uppercase tracking-tight text-white font-mono">
            ¿Cómo funciona router-core?
          </h2>
          <p className="mt-2 text-sm text-neutral-300 leading-relaxed max-w-3xl">
            Los routers domésticos viejos (como el TP-Link WR841N) tienen interfaces web obsoletas, lentas y sin APIs. En lugar de arriesgarse a flashear firmware o usar scrapers inestables, <strong className="text-white">router-core</strong> levanta un servicio local en <code className="text-primary font-mono font-bold">127.0.0.1:8484</code> que expone el estado del router como JSON tipado y permite que un agente <strong className="text-white">MiniMax M3</strong> en GMI Cloud audite la seguridad en lenguaje natural.
          </p>
        </div>

        {/* 4 Invariantes Clave (Sin cards pesadas) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 pt-2 font-sans text-sm">
          <div className="border-l-2 border-emerald-500 pl-4 space-y-1.5">
            <div className="font-mono font-bold text-xs uppercase text-white flex items-center gap-2">
              <Eye className="h-4 w-4 text-emerald-400" />
              <span>100% Solo Lectura (GET Only)</span>
            </div>
            <p className="text-xs text-neutral-400 leading-relaxed">
              La mutación (<code className="text-emerald-400 font-mono">CapMutate</code>) es inexistente en el sistema de tipos. Es imposible reiniciar o desconfigurar el router.
            </p>
          </div>

          <div className="border-l-2 border-primary pl-4 space-y-1.5">
            <div className="font-mono font-bold text-xs uppercase text-white flex items-center gap-2">
              <Cpu className="h-4 w-4 text-primary" />
              <span>Razonamiento con MiniMax M3</span>
            </div>
            <p className="text-xs text-neutral-400 leading-relaxed">
              Usa 1M de contexto en GMI Cloud para interpretar tablas ARP, cifrado Wi-Fi y puertos con fallback automático a MiniMax M2.7.
            </p>
          </div>

          <div className="border-l-2 border-amber-500 pl-4 space-y-1.5">
            <div className="font-mono font-bold text-xs uppercase text-white flex items-center gap-2">
              <Lock className="h-4 w-4 text-amber-400" />
              <span>Aislamiento RFC1918 &amp; Loopback</span>
            </div>
            <p className="text-xs text-neutral-400 leading-relaxed">
              Rechaza IPs públicas de Internet y dominios externos. Inmune a ataques SSRF y DNS rebinding.
            </p>
          </div>

          <div className="border-l-2 border-cyan-500 pl-4 space-y-1.5">
            <div className="font-mono font-bold text-xs uppercase text-white flex items-center gap-2">
              <Server className="h-4 w-4 text-cyan-400" />
              <span>Estados Honestos (First-Class Unknown)</span>
            </div>
            <p className="text-xs text-neutral-400 leading-relaxed">
              Nunca inventa datos; reporta explícitamente si un parámetro está <em>verified</em>, <em>absent</em> o <em>unsupported</em>.
            </p>
          </div>
        </div>
      </section>

      {/* 4. COMPARATIVA DE SOLUCIONES */}
      <section className="border-t-2 border-neutral-800 pt-12 space-y-6">
        <div>
          <h2 className="text-2xl font-black uppercase tracking-tight text-white font-mono">
            Comparativa Frente a Otras Soluciones
          </h2>
          <p className="mt-1 text-sm text-neutral-400">
            Por qué router-core es la alternativa más segura y rápida para hardware legado.
          </p>
        </div>

        <div className="overflow-x-auto border-2 border-neutral-800 bg-black font-mono text-xs">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b-2 border-neutral-800 bg-neutral-900 text-neutral-300">
                <th className="p-3.5 font-bold uppercase">Criterio</th>
                <th className="p-3.5 font-bold uppercase text-neutral-400">Web Scraping</th>
                <th className="p-3.5 font-bold uppercase text-neutral-400">OpenWrt / DD-WRT</th>
                <th className="p-3.5 font-bold uppercase text-primary bg-primary/10 border-l-2 border-primary">router-core + MiniMax M3</th>
              </tr>
            </thead>
            <tbody className="divide-y border-neutral-800 text-neutral-300">
              <tr>
                <td className="p-3.5 font-sans font-semibold text-white">Riesgo de Daño al Router</td>
                <td className="p-3.5 text-rose-400">Medio (POST inestables)</td>
                <td className="p-3.5 text-rose-400">Alto (Flasheo de ROM)</td>
                <td className="p-3.5 text-emerald-400 font-bold bg-primary/5 border-l-2 border-primary">Cero (GET Only forzado)</td>
              </tr>
              <tr>
                <td className="p-3.5 font-sans font-semibold text-white">Auditoría con IA</td>
                <td className="p-3.5 text-neutral-500">No</td>
                <td className="p-3.5 text-neutral-500">No</td>
                <td className="p-3.5 text-emerald-400 font-bold bg-primary/5 border-l-2 border-primary">MiniMax M3 (1M Context)</td>
              </tr>
              <tr>
                <td className="p-3.5 font-sans font-semibold text-white">Instalación</td>
                <td className="p-3.5 text-neutral-400">Scripts manuales</td>
                <td className="p-3.5 text-rose-400">Desarmar / Serial</td>
                <td className="p-3.5 text-emerald-400 font-bold bg-primary/5 border-l-2 border-primary">1 Comando cURL o App GUI</td>
              </tr>
              <tr>
                <td className="p-3.5 font-sans font-semibold text-white">Manejo de Errores</td>
                <td className="p-3.5 text-rose-400">Inventa falsos false</td>
                <td className="p-3.5 text-neutral-400">Depende de la distro</td>
                <td className="p-3.5 text-emerald-400 font-bold bg-primary/5 border-l-2 border-primary">First-Class Unknown (Honesto)</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* 5. FOOTER / HACKATHON INFO & CTA FINAL */}
      <footer className="border-t-2 border-neutral-800 pt-10 pb-6 space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="space-y-1">
            <div className="font-mono font-bold text-sm text-white">router-core</div>
            <p className="text-xs text-neutral-400">
              Proyecto creado para la hackathon <strong className="text-white">GMI Cloud × MiniMax Week 2026</strong>.
            </p>
          </div>

          <div className="flex items-center gap-3 font-mono text-xs">
            <Button
              variant="default"
              onClick={() => onOpenDashboard && onOpenDashboard()}
              className="h-10 px-4 gap-2"
            >
              <Zap className="h-3.5 w-3.5" />
              <span>Abrir Asistente</span>
            </Button>

            <a
              href="https://github.com/Quiarom/router-core"
              target="_blank"
              rel="noreferrer"
              className="h-10 px-4 bg-neutral-900 hover:bg-neutral-800 border border-neutral-700 text-neutral-300 flex items-center gap-1.5 transition-colors"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              <span>Ver Repo</span>
            </a>
          </div>
        </div>
      </footer>

    </div>
  );
}

export default LandingPage;
