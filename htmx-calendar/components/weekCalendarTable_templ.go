// Code generated by templ - DO NOT EDIT.

// templ: version: v0.3.906
package components

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import (
	"htmx-calendar/components/shared"
	"htmx-calendar/models"
	"strconv"
	"strings"
	"time"
)

func weekCalendarTable(calendarData [][7]time.Time, eventsData []models.EventData, currentMonthAndYear time.Time, from string, week int) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		days := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
		heightClass := "h-8 lg:h-14 xl:h-16"
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 1, "<table class=\"w-full h-full table-fixed\" style=\"view-transition-name:none\"><tbody style=\"view-transition-name:none\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		for idx, day := range days {
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 2, "<tr>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			td1Class := generateWeekTdClass(idx, 0, from)
			td2Class := generateWeekTdClass(idx, 1, from)
			var templ_7745c5c3_Var2 = []any{"border-r-2 border-gray-300 font-semibold text-center", td1Class}
			templ_7745c5c3_Err = templ.RenderCSSItems(ctx, templ_7745c5c3_Buffer, templ_7745c5c3_Var2...)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 3, "<td class=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var3 string
			templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(templ.CSSClasses(templ_7745c5c3_Var2).String())
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 1, Col: 0}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 4, "\"><span style=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var4 string
			templ_7745c5c3_Var4, templ_7745c5c3_Err = templruntime.SanitizeStyleAttributeValues("view-transition-name:headerDay-" + day)
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 21, Col: 59}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var4))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 5, "\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var5 string
			templ_7745c5c3_Var5, templ_7745c5c3_Err = templ.JoinStringErrs(day)
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 21, Col: 67}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var5))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 6, "</span></td>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var6 = []any{"w-11/12 border-r-2 border-gray-300", td2Class}
			templ_7745c5c3_Err = templ.RenderCSSItems(ctx, templ_7745c5c3_Buffer, templ_7745c5c3_Var6...)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 7, "<td class=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var7 string
			templ_7745c5c3_Var7, templ_7745c5c3_Err = templ.JoinStringErrs(templ.CSSClasses(templ_7745c5c3_Var6).String())
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 1, Col: 0}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var7))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 8, "\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			displayAll := displayDateAndData(calendarData[0][idx], currentMonthAndYear)
			active, activeClass := getActiveFlagAndActiveClassWeekTd(calendarData[0][idx], currentMonthAndYear)
			dndClassBindExpression := "$store.data.checkDnDRowCol(" + strconv.Itoa(idx) + "," + strconv.Itoa(1) + ")?'border-3 border-teal-600':'border-3 border-transparent'"
			allowDnD := calendarData[0][idx].Sub(time.Now()).Hours() >= -24 && displayAll
			templ_7745c5c3_Err = templ.RenderScriptItems(ctx, templ_7745c5c3_Buffer, templ.JSFuncCall("setAlpineStoreDnDRowCol", templ.JSExpression("event"), idx, 1, allowDnD), templ.JSFuncCall("removeAlpineStoreDnDRowCol", idx, 1), templ.JSFuncCall("eventDropped", templ.JSExpression("event"), calendarData[0][idx].Format("2006-01-02"), allowDnD))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 9, "<div class=\"p-1 flex flex-col w-full h-full justify-start items-start gap-1 border-dashed\" x-bind:class=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var8 string
			templ_7745c5c3_Var8, templ_7745c5c3_Err = templ.JoinStringErrs(dndClassBindExpression)
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 30, Col: 44}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var8))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 10, "\" ondragover=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var9 templ.ComponentScript = templ.JSFuncCall("setAlpineStoreDnDRowCol", templ.JSExpression("event"), idx, 1, allowDnD)
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ_7745c5c3_Var9.Call)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 11, "\" ondragleave=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var10 templ.ComponentScript = templ.JSFuncCall("removeAlpineStoreDnDRowCol", idx, 1)
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ_7745c5c3_Var10.Call)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 12, "\" ondrop=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var11 templ.ComponentScript = templ.JSFuncCall("eventDropped", templ.JSExpression("event"), calendarData[0][idx].Format("2006-01-02"), allowDnD)
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ_7745c5c3_Var11.Call)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 13, "\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			if displayAll {
				templ_7745c5c3_Err = shared.AddEventLink(calendarData[0][idx], week, active).Render(ctx, templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
			}
			var templ_7745c5c3_Var12 = []any{"w-full overflow-x-hidden overflow-y-auto flex flex-col gap-1 rounded p-1 scrollbar-thin scrollbar-track-purple-300 scrollbar-thumb-purple-600 focus:outline-2 focus:outline-purple-600", heightClass}
			templ_7745c5c3_Err = templ.RenderCSSItems(ctx, templ_7745c5c3_Buffer, templ_7745c5c3_Var12...)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 14, "<div class=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var13 string
			templ_7745c5c3_Var13, templ_7745c5c3_Err = templ.JoinStringErrs(templ.CSSClasses(templ_7745c5c3_Var12).String())
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 1, Col: 0}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var13))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 15, "\" style=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var14 string
			templ_7745c5c3_Var14, templ_7745c5c3_Err = templruntime.SanitizeStyleAttributeValues("view-transition-name:items-" + calendarData[0][idx].Format("2006-01-02"))
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 40, Col: 89}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var14))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 16, "\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			if displayAll {
				for _, event := range filterDateEvents(eventsData, calendarData[0][idx]) {
					var templ_7745c5c3_Var15 = []any{"group text-white text-sm cursor-pointer py-1 px-3 rounded w-full truncate shrink-0 flex items-center gap-2 lg:text-base", activeClass}
					templ_7745c5c3_Err = templ.RenderCSSItems(ctx, templ_7745c5c3_Buffer, templ_7745c5c3_Var15...)
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					templ_7745c5c3_Err = templ.RenderScriptItems(ctx, templ_7745c5c3_Buffer, templ.JSFuncCall("dragStarted", templ.JSExpression("event"), event.Id))
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 17, "<div draggable=\"")
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					var templ_7745c5c3_Var16 string
					templ_7745c5c3_Var16, templ_7745c5c3_Err = templ.JoinStringErrs("true")
					if templ_7745c5c3_Err != nil {
						return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 45, Col: 29}
					}
					_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var16))
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 18, "\" data-id=\"")
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					var templ_7745c5c3_Var17 string
					templ_7745c5c3_Var17, templ_7745c5c3_Err = templ.JoinStringErrs(event.Id)
					if templ_7745c5c3_Err != nil {
						return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 46, Col: 29}
					}
					_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var17))
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 19, "\" ondragstart=\"")
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					var templ_7745c5c3_Var18 templ.ComponentScript = templ.JSFuncCall("dragStarted", templ.JSExpression("event"), event.Id)
					_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ_7745c5c3_Var18.Call)
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 20, "\" class=\"")
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					var templ_7745c5c3_Var19 string
					templ_7745c5c3_Var19, templ_7745c5c3_Err = templ.JoinStringErrs(templ.CSSClasses(templ_7745c5c3_Var15).String())
					if templ_7745c5c3_Err != nil {
						return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 1, Col: 0}
					}
					_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var19))
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 21, "\" style=\"view-transition-name:none\"><span class=\"flex-1 truncate\">")
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					var templ_7745c5c3_Var20 string
					templ_7745c5c3_Var20, templ_7745c5c3_Err = templ.JoinStringErrs(event.Task)
					if templ_7745c5c3_Err != nil {
						return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 51, Col: 53}
					}
					_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var20))
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 22, "</span> ")
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
					if active {
						templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 23, "<button name=\"eventId\" value=\"")
						if templ_7745c5c3_Err != nil {
							return templ_7745c5c3_Err
						}
						var templ_7745c5c3_Var21 string
						templ_7745c5c3_Var21, templ_7745c5c3_Err = templ.JoinStringErrs(event.Id)
						if templ_7745c5c3_Err != nil {
							return templ.Error{Err: templ_7745c5c3_Err, FileName: `components/weekCalendarTable.templ`, Line: 55, Col: 29}
						}
						_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var21))
						if templ_7745c5c3_Err != nil {
							return templ_7745c5c3_Err
						}
						templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 24, "\" hx-delete=\"/delete\" hx-confirm=\"Are you sure you want to delete this event?\" hx-target=\"closest div\" hx-swap=\"outerHTML\" class=\"appearance-none outline-none p-1 cursor-pointer opacity-90 group-hover:opacity-100 transition duration-300 rounded-full focus:opacity-100 focus:ring-1 focus:ring-white disabled:cursor-not-allowed disabled:opacity-50\"><svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 24 24\" fill=\"currentColor\" class=\"size-5\"><path fill-rule=\"evenodd\" d=\"M16.5 4.478v.227a48.816 48.816 0 0 1 3.878.512.75.75 0 1 1-.256 1.478l-.209-.035-1.005 13.07a3 3 0 0 1-2.991 2.77H8.084a3 3 0 0 1-2.991-2.77L4.087 6.66l-.209.035a.75.75 0 0 1-.256-1.478A48.567 48.567 0 0 1 7.5 4.705v-.227c0-1.564 1.213-2.9 2.816-2.951a52.662 52.662 0 0 1 3.369 0c1.603.051 2.815 1.387 2.815 2.951Zm-6.136-1.452a51.196 51.196 0 0 1 3.273 0C14.39 3.05 15 3.684 15 4.478v.113a49.488 49.488 0 0 0-6 0v-.113c0-.794.609-1.428 1.364-1.452Zm-.355 5.945a.75.75 0 1 0-1.5.058l.347 9a.75.75 0 1 0 1.499-.058l-.346-9Zm5.48.058a.75.75 0 1 0-1.498-.058l-.347 9a.75.75 0 0 0 1.5.058l.345-9Z\" clip-rule=\"evenodd\"></path></svg></button>")
						if templ_7745c5c3_Err != nil {
							return templ_7745c5c3_Err
						}
					}
					templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 25, "</div>")
					if templ_7745c5c3_Err != nil {
						return templ_7745c5c3_Err
					}
				}
			}
			templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 26, "</div></div></td></tr>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 27, "</tbody></table>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

func generateWeekTdClass(row int, col int, from string) string {
	animationClass := ""
	if strings.ToLower(from) == "prev" {
		animationClass = "animate-slide-right"
	} else if strings.ToLower(from) == "next" {
		animationClass = "animate-slide-left"
	}
	borderBottomClass := "border-b-2 "
	if row == 6 {
		borderBottomClass = ""
	} else if row == 0 {
		borderBottomClass = "border-y-2 "
	}
	if col != 0 {
		borderBottomClass += animationClass
	}
	return borderBottomClass
}

func getActiveFlagAndActiveClassWeekTd(tdData time.Time, currentMonthAndYear time.Time) (bool, string) {
	active := true
	activeClass := "bg-orange-600"
	if tdData.Month() != currentMonthAndYear.Month() || tdData.Year() != currentMonthAndYear.Year() {
		active = false
		activeClass = "bg-orange-500"
	}
	if tdData.Sub(time.Now()).Hours() < -24 {
		activeClass = "bg-orange-500"
		active = false
	}
	return active, activeClass
}

func displayDateAndData(tdData time.Time, currentMonthAndYear time.Time) bool {
	if tdData.Month() > currentMonthAndYear.Month() || tdData.Year() > currentMonthAndYear.Year() {
		return false
	}
	return true
}

var _ = templruntime.GeneratedTemplate
