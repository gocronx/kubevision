import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { AlertTriangle, FlaskConical, Plus, Save, Search, Trash2 } from "lucide-react"
import { toast } from "sonner"
import api from "@/lib/api"
import { useAuth } from "@/stores/auth-store"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"

type Mapping = { id?: number; groupId: string; role: string; priority: number }
type Settings = {
  enabled: boolean; serverUrl: string; startTls: boolean; allowPlaintext: boolean; caBundle: string
  bindDn: string; bindPassword: string; credentialConfigured?: boolean; connectTimeoutSecs: number
  searchTimeoutSecs: number; userBaseDn: string; userFilter: string; stableIdAttribute: string
  usernameAttribute: string; displayAttribute: string; emailAttribute: string; groupAttribute: string
  fallbackRole: string; refreshMapping: boolean; mappings: Mapping[]
}

const defaults: Settings = { enabled:false, serverUrl:"", startTls:true, allowPlaintext:false, caBundle:"", bindDn:"", bindPassword:"", connectTimeoutSecs:5, searchTimeoutSecs:8, userBaseDn:"", userFilter:"(uid={{username}})", stableIdAttribute:"entryUUID", usernameAttribute:"uid", displayAttribute:"displayName", emailAttribute:"mail", groupAttribute:"memberOf", fallbackRole:"viewer", refreshMapping:true, mappings:[] }
const roles = ["viewer", "editor", "admin"]
const attributeKeys = ["stableIdAttribute", "usernameAttribute", "displayAttribute", "emailAttribute", "groupAttribute"] as const

export function DirectorySettingsPage() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const [form, setForm] = useState<Settings>(defaults)
  const [loadError, setLoadError] = useState(false)
  const [previewID, setPreviewID] = useState("")
  const [preview, setPreview] = useState<{ groups:string[]; role:string; matchedRule?:Mapping } | null>(null)
  const admin = user?.role === "admin" || user?.role === "super-admin"

  const loadSettings = useCallback(async () => {
    if (!admin) return
    try {
      const value = await api.get("/directory/config") as unknown as Partial<Settings>
      setForm({
        ...defaults,
        ...value,
        mappings: Array.isArray(value.mappings) ? value.mappings : [],
        bindPassword: "",
      })
      setLoadError(false)
    } catch {
      setLoadError(true)
    }
  }, [admin])

  useEffect(() => {
    if (!admin) return
    let active = true
    api.get("/directory/config").then((value) => {
      if (!active) return
      const settings = value as unknown as Partial<Settings>
      setForm({
        ...defaults,
        ...settings,
        mappings: Array.isArray(settings.mappings) ? settings.mappings : [],
        bindPassword: "",
      })
      setLoadError(false)
    }).catch(() => {
      if (active) setLoadError(true)
    })
    return () => { active = false }
  }, [admin])
  if (!admin) return <p className="p-6 text-sm text-muted-foreground">{t("directory.adminRequired")}</p>

  const set = <K extends keyof Settings>(key:K, value:Settings[K]) => setForm((old) => ({ ...old, [key]:value }))
  const save = async () => { await api.put("/directory/config", form); toast.success(t("directory.savedToast")) }
  const test = async () => {
    const result = await api.post("/directory/test", form) as {ok:boolean;category:string}
    toast[result.ok ? "success" : "error"](result.ok ? t("directory.testSucceeded") : t("directory.testFailed", { category: result.category }))
  }
  const runPreview = async () => setPreview(await api.post("/directory/preview", {identifier:previewID}) as typeof preview)
  const roleLabels = Object.fromEntries(roles.map((role) => [role, t(`directory.role.${role}`)]))

  return <div className="mx-auto flex max-w-5xl flex-col gap-6 p-6">
    <div><h1 className="text-2xl font-semibold">{t("directory.title")}</h1><p className="text-sm text-muted-foreground">{t("directory.description")}</p></div>
    {loadError && <div role="alert" className="flex items-center justify-between gap-3 rounded-md border border-destructive/50 bg-destructive/5 p-3 text-sm text-destructive"><span>{t("directory.loadFailed")}</span><Button size="sm" variant="outline" onClick={loadSettings}>{t("directory.retry")}</Button></div>}
    {form.allowPlaintext && <div className="flex items-center gap-2 rounded-md border border-amber-500 bg-amber-50 p-3 text-sm text-amber-900"><AlertTriangle className="size-4"/>{t("directory.plaintextWarning")}</div>}
    <Card><CardHeader><CardTitle className="text-base">{t("directory.connection")}</CardTitle></CardHeader><CardContent className="grid gap-4 md:grid-cols-2">
      <Check label={t("directory.enabled")} checked={form.enabled} onChange={(v)=>set("enabled",v)}/><Check label={t("directory.startTls")} checked={form.startTls} onChange={(v)=>set("startTls",v)}/><Check label={t("directory.allowPlaintext")} checked={form.allowPlaintext} onChange={(v)=>set("allowPlaintext",v)}/><Check label={t("directory.refreshMapping")} checked={form.refreshMapping} onChange={(v)=>set("refreshMapping",v)}/>
      <Field label={t("directory.serverUrl")} value={form.serverUrl} onChange={(v)=>set("serverUrl",v)} placeholder="ldaps://directory.example.com:636"/><Field label={t("directory.bindDn")} value={form.bindDn} onChange={(v)=>set("bindDn",v)}/><Field label={`${t("directory.bindPassword")}${form.credentialConfigured ? ` (${t("directory.saved")})` : ""}`} value={form.bindPassword} onChange={(v)=>set("bindPassword",v)} password/><Field label={t("directory.userBaseDn")} value={form.userBaseDn} onChange={(v)=>set("userBaseDn",v)}/><Field label={t("directory.userFilter")} value={form.userFilter} onChange={(v)=>set("userFilter",v)}/><Field label={t("directory.fallbackRole")} value={form.fallbackRole} onChange={(v)=>set("fallbackRole",v)} options={roles} optionLabels={roleLabels}/>
      <div className="md:col-span-2"><Label>{t("directory.caBundle")}</Label><Textarea value={form.caBundle} onChange={(e)=>set("caBundle",e.target.value)} className="mt-1 font-mono text-xs"/></div>
    </CardContent></Card>
    <Card><CardHeader><CardTitle className="text-base">{t("directory.attributes")}</CardTitle></CardHeader><CardContent className="grid gap-4 md:grid-cols-3">{attributeKeys.map((key)=><Field key={key} label={t(`directory.attribute.${key}`)} value={form[key]} onChange={(v)=>set(key,v)}/>)}</CardContent></Card>
    <Card><CardHeader className="flex-row items-center justify-between"><CardTitle className="text-base">{t("directory.mappings")}</CardTitle><Button size="sm" variant="outline" onClick={()=>set("mappings",[...form.mappings,{groupId:"",role:"viewer",priority:(form.mappings.at(-1)?.priority ?? 0)+10}])}><Plus/>{t("directory.addMapping")}</Button></CardHeader><CardContent className="space-y-2">{form.mappings.map((mapping,index)=><div key={index} className="grid grid-cols-[1fr_9rem_7rem_2.5rem] gap-2"><Input value={mapping.groupId} placeholder={t("directory.groupIdentifier")} onChange={(e)=>set("mappings",form.mappings.map((m,i)=>i===index?{...m,groupId:e.target.value}:m))}/><select className="rounded-md border bg-background px-2 text-sm" value={mapping.role} onChange={(e)=>set("mappings",form.mappings.map((m,i)=>i===index?{...m,role:e.target.value}:m))}>{roles.map((role)=><option key={role} value={role}>{roleLabels[role]}</option>)}</select><Input type="number" value={mapping.priority} onChange={(e)=>set("mappings",form.mappings.map((m,i)=>i===index?{...m,priority:Number(e.target.value)}:m))}/><Button size="icon" variant="ghost" title={t("directory.removeMapping")} onClick={()=>set("mappings",form.mappings.filter((_,i)=>i!==index))}><Trash2/></Button></div>)}</CardContent></Card>
    <Card><CardHeader><CardTitle className="text-base">{t("directory.previewTitle")}</CardTitle></CardHeader><CardContent className="space-y-3"><div className="flex gap-2"><Input value={previewID} onChange={(e)=>setPreviewID(e.target.value)} placeholder={t("directory.userIdentifier")}/><Button variant="outline" onClick={runPreview} disabled={!previewID}><Search/>{t("directory.preview")}</Button></div>{preview&&<div className="rounded-md border p-3 text-sm"><p>{t("directory.resultingRole")}<strong>{roleLabels[preview.role] ?? preview.role}</strong></p><p className="mt-1 text-muted-foreground">{t("directory.groups")}{preview.groups.join(", ") || t("directory.none")}</p></div>}</CardContent></Card>
    <div className="flex justify-end gap-2"><Button variant="outline" onClick={test}><FlaskConical/>{t("directory.test")}</Button><Button onClick={save}><Save/>{t("directory.save")}</Button></div>
  </div>
}

function Field({label,value,onChange,placeholder,password,options,optionLabels}:{label:string;value:string|number;onChange:(value:string)=>void;placeholder?:string;password?:boolean;options?:string[];optionLabels?:Record<string,string>}) { return <div><Label>{label}</Label>{options?<select className="mt-1 h-9 w-full rounded-md border bg-background px-3 text-sm" value={value} onChange={(e)=>onChange(e.target.value)}>{options.map((option)=><option key={option} value={option}>{optionLabels?.[option] ?? option}</option>)}</select>:<Input className="mt-1" type={password?"password":"text"} value={value} placeholder={placeholder} onChange={(e)=>onChange(e.target.value)}/>}</div> }
function Check({label,checked,onChange}:{label:string;checked:boolean;onChange:(value:boolean)=>void}) { return <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={checked} onChange={(e)=>onChange(e.target.checked)} className="size-4"/>{label}</label> }
