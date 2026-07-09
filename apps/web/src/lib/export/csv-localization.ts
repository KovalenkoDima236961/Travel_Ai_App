import type { SupportedLanguage } from "@/lib/i18n/languages";

const LABELS: Record<Exclude<SupportedLanguage, "en">, Record<string, string>> = {
  es: {
    Summary: "Resumen", Metric: "Métrica", Value: "Valor", Currency: "Moneda",
    Day: "Día", Date: "Fecha", Time: "Hora", Item: "Elemento", Cost: "Coste",
    Name: "Nombre", Type: "Tipo", Amount: "Importe", Warning: "Aviso",
    Warnings: "Avisos", "Estimated total": "Total estimado", "Cost by day": "Coste por día",
    "Cost by category": "Coste por categoría", "Cost by source": "Coste por fuente",
    "Expensive items": "Elementos costosos"
  },
  uk: {
    Summary: "Підсумок", Metric: "Показник", Value: "Значення", Currency: "Валюта",
    Day: "День", Date: "Дата", Time: "Час", Item: "Елемент", Cost: "Вартість",
    Name: "Назва", Type: "Тип", Amount: "Сума", Warning: "Попередження",
    Warnings: "Попередження", "Estimated total": "Орієнтовна сума",
    "Cost by day": "Витрати за днями", "Cost by category": "Витрати за категоріями",
    "Cost by source": "Витрати за джерелами", "Expensive items": "Найдорожчі елементи"
  },
  fr: {
    Summary: "Résumé", Metric: "Indicateur", Value: "Valeur", Currency: "Devise",
    Day: "Jour", Date: "Date", Time: "Heure", Item: "Élément", Cost: "Coût",
    Name: "Nom", Type: "Type", Amount: "Montant", Warning: "Avertissement",
    Warnings: "Avertissements", "Estimated total": "Total estimé",
    "Cost by day": "Coût par jour", "Cost by category": "Coût par catégorie",
    "Cost by source": "Coût par source", "Expensive items": "Éléments coûteux"
  }
};

export function localizeCsvText(csv: string, language: SupportedLanguage): string {
  if (language === "en") {
    return csv;
  }
  const labels = LABELS[language];
  return csv
    .split("\n")
    .map((line) => {
      if (labels[line]) {
        return labels[line];
      }
      return line
        .split(",")
        .map((cell) => labels[cell] ?? cell)
        .join(",");
    })
    .join("\n");
}
