import React, { PureComponent } from "react"
import { ReactComponent as LogoWordmarkSvg } from "./assets/svg/logo-wordmark-gray.svg"
import AnsiLine from "./AnsiLine"
import TimeAgo from "react-timeago"
import "./ErrorPane.scss"
import { zeroTime } from "./time"
import { Build } from "./types"

class ErrorResource {
  public name: string
  public buildHistory: Array<Build>
  public resourceInfo: ResourceInfo

  constructor(resource: any) {
    this.name = resource.Name
    this.buildHistory = resource.BuildHistory
    if (resource.ResourceInfo) {
      this.resourceInfo = {
        podCreationTime: resource.ResourceInfo.PodCreationTime,
        podStatus: resource.ResourceInfo.PodStatus,
        podRestarts: resource.ResourceInfo.PodRestarts,
        podLog: resource.ResourceInfo.PodLog,
      }
    } else {
      this.resourceInfo = {
        podCreationTime: zeroTime,
        podStatus: "",
        podRestarts: 0,
        podLog: "",
      }
    }
  }
}

type ResourceInfo = {
  podCreationTime: string
  podStatus: string
  podRestarts: number
  podLog: string
}

type ErrorsProps = {
  resources: Array<ErrorResource>
}

class ErrorPane extends PureComponent<ErrorsProps> {
  render() {
    let el = (
      <section className="Pane-empty-message">
        <LogoWordmarkSvg />
        <h2>No Errors Found</h2>
      </section>
    )
    let errorElements: Array<JSX.Element> = []
    this.props.resources.forEach(r => {
      if (
        r.resourceInfo.podStatus === "Error" ||
        r.resourceInfo.podStatus === "CrashLoopBackOff"
      ) {
        errorElements.push(
          <li key={"resourceInfoError" + r.name} className="ErrorPane-item">
            <header>
              <p>{r.name}</p>
              <p>{r.resourceInfo.podCreationTime}</p>
            </header>
            <section>{r.resourceInfo.podLog}</section>
          </li>
        )
      } else if (r.resourceInfo.podRestarts > 0) {
        errorElements.push(
          <li key={"resourceInfoPodCrash" + r.name} className="ErrorPane-item">
            <header>
              <p>{r.name}</p>
              <p>{`Restarts: ${r.resourceInfo.podRestarts}`}</p>
              <p>{r.resourceInfo.podCreationTime}</p>
            </header>
            <section>
              <p>{`Last log line: ${r.resourceInfo.podLog}`}</p>
            </section>
          </li>
        )
      }
      if (r.buildHistory.length > 0) {
        let lastBuild = r.buildHistory[0]
        if (lastBuild.Error !== null) {
          errorElements.push(
            <li key={"buildError" + r.name} className="ErrorPane-item">
              <header>
                <p>{r.name}</p>
                <p>
                  <TimeAgo date={lastBuild.FinishTime} />
                </p>
              </header>
              <section>
                {lastBuild.Log.split("\n").map((l, i) => (
                  <AnsiLine key={"logLine" + i} line={l} />
                ))}
              </section>
            </li>
          )
        }
      }
    })

    if (errorElements.length > 0) {
      el = <ul>{errorElements}</ul>
    }

    return <section className="ErrorPane">{el}</section>
  }
}

export default ErrorPane
export { ErrorResource }
