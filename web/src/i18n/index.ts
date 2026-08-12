import i18n from "i18next"
import { initReactI18next } from "react-i18next"
import en from "./en.json"
import zh from "./zh.json"

const savedLang = localStorage.getItem("language")
const initialLang = savedLang === "en" || savedLang === "zh"
  ? savedLang
  : navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en"

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    zh: { translation: zh },
  },
  lng: initialLang,
  supportedLngs: ["en", "zh"],
  fallbackLng: "en",
  interpolation: {
    escapeValue: false,
  },
})

export default i18n
